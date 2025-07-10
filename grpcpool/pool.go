package grpcpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var (
	// ErrPoolClosed is returned from the connection pool is closed.
	ErrPoolClosed = errors.New("grpcpool: connection pool is closed")

	// ErrPoolExhausted is returned from a pool connection method
	// when the maximum number of connections in the pool has been reached.
	ErrPoolExhausted = errors.New("grpcpool: connection pool exhausted")

	// ErrPoolTimeout timed out waiting to get a connection from the connection pool.
	ErrPoolTimeout = errors.New("grpcpool: connection pool timeout")

	// ErrConnTimeout timed out waiting to get a connection from the connection pool.
	ErrConnTimeout = errors.New("grpcpool: connection timeout")
)

type ConnPool struct {
	opts *Options

	idleConns   chan *Conn
	reqChan     chan struct{}
	activeCount int32
	closed      uint32
	mu          sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	stats          Stats
	waitDurationNs atomic.Int64

	timers *TimerPool
}

func NewConnPool(ctx context.Context, opts ...Option) (*ConnPool, error) {
	opt := NewOptions(opts...)

	if opt.DialFunc == nil {
		return nil, fmt.Errorf("DialFunc must not be nil")
	}

	ctx, cancel := context.WithCancel(ctx)

	p := &ConnPool{
		opts:      opt,
		idleConns: make(chan *Conn, opt.MaxIdleConns),
		reqChan:   make(chan struct{}, opt.PoolSize*2),
		ctx:       ctx,
		cancel:    cancel,
		timers:    NewTimerPool(),
	}

	for i := 0; i < opt.MinIdleConns; i++ {
		if err := p.newIdleConn(); err != nil {
			p.Close()
			return nil, fmt.Errorf("failed to create idle connection: %w", err)
		}
	}

	p.wg.Add(1)
	go p.checkIdleConns()

	return p, nil
}

func (p *ConnPool) Close() error {
	if !atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		return ErrPoolClosed
	}

	p.cancel()
	p.wg.Wait()

	close(p.idleConns)
	close(p.reqChan)

	for conn := range p.idleConns {
		p.closeConn(conn)
	}
	return nil
}

func (p *ConnPool) Stats() *Stats {
	return &Stats{
		HitCount:     atomic.LoadUint32(&p.stats.HitCount),
		MissCount:    atomic.LoadUint32(&p.stats.MissCount),
		TimeoutCount: atomic.LoadUint32(&p.stats.TimeoutCount),
		WaitCount:    atomic.LoadUint32(&p.stats.WaitCount),

		TotalConns: uint32(atomic.LoadInt32(&p.activeCount)),
		IdleConns:  uint32(len(p.idleConns)),
		StaleConns: atomic.LoadUint32(&p.stats.StaleConns),
	}
}

func (p *ConnPool) Get(ctx context.Context) (*Conn, error) {
	if p.isClosed() {
		return nil, ErrPoolClosed
	}

	if err := p.waitTurn(ctx); err != nil {
		return nil, err
	}

out:
	for {
		select {
		case conn := <-p.idleConns:
			if p.isConnAlive(conn) {
				conn.lastUsed = time.Now()
				atomic.AddUint32(&p.stats.HitCount, 1)
				return conn, nil
			}

			atomic.AddUint32(&p.stats.StaleConns, 1)
			p.closeConn(conn)
			p.freeTurn()

		default:
			break out
		}
	}

	atomic.AddUint32(&p.stats.MissCount, 1)

	conn, err := p.dialConn()
	if err != nil {
		p.freeTurn()
		return nil, err
	}
	return conn, nil
}

func (p *ConnPool) Put(conn *Conn) {
	if p.isClosed() || !p.isConnAlive(conn) {
		p.closeConn(conn)
		return
	}

	conn.lastUsed = time.Now()
	p.freeTurn()

	p.put(conn)
}

func (p *ConnPool) put(conn *Conn) {
	select {
	case p.idleConns <- conn:
	default:
		p.closeConn(conn)
	}
}

func (p *ConnPool) waitTurn(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case p.reqChan <- struct{}{}:
		return nil
	default:
	}

	start := time.Now()
	timer := p.timers.Get(p.opts.PoolTimeout)
	defer p.timers.Put(timer)

	select {
	case <-ctx.Done():
		return ctx.Err()

	case p.reqChan <- struct{}{}:
		p.waitDurationNs.Add(time.Since(start).Nanoseconds())
		atomic.AddUint32(&p.stats.WaitCount, 1)
		return nil

	case <-timer.C:
		atomic.AddUint32(&p.stats.TimeoutCount, 1)
		return ErrPoolTimeout
	}
}

func (p *ConnPool) freeTurn() {
	select {
	case <-p.reqChan:
	case <-p.ctx.Done():
	}
}

func (p *ConnPool) checkIdleConns() {
	ticker := time.NewTicker(p.opts.IdleCheckFreq)
	defer ticker.Stop()

out:
	for {
		select {
		case <-ticker.C:
			p.cleanExpiredConns()
			p.adjustIdleConns()

		case <-p.ctx.Done():
			break out
		}
	}
	p.wg.Done()
}

func (p *ConnPool) cleanExpiredConns() {
	maxBatchSize := 50

	if len(p.idleConns) < maxBatchSize {
		maxBatchSize = len(p.idleConns)
	}

	for i := 0; i <= maxBatchSize; i++ {
		select {
		case conn := <-p.idleConns:
			if p.isConnAlive(conn) {
				p.put(conn)
			} else {
				atomic.AddUint32(&p.stats.StaleConns, 1)
				p.closeConn(conn)
			}

		default:
			return
		}
	}
}

func (p *ConnPool) adjustIdleConns() {
	idleCount := len(p.idleConns)

	if idleCount < p.opts.MinIdleConns {
		needed := p.opts.MinIdleConns - idleCount
		for i := 0; i < needed; i++ {
			select {
			case p.reqChan <- struct{}{}:
				if err := p.newIdleConn(); err != nil {
					break
				}
				p.freeTurn()

			default:
				return
			}
		}
		return
	}

	if idleCount > p.opts.MaxIdleConns {
		excess := idleCount - p.opts.MaxIdleConns
		for i := 0; i < excess; i++ {
			select {
			case conn := <-p.idleConns:
				atomic.AddUint32(&p.stats.StaleConns, 1)
				p.closeConn(conn)
			default:
				return
			}
		}
	}
}

func (p *ConnPool) newIdleConn() error {
	conn, err := p.dialConn()
	if err != nil {
		return err
	}

	p.put(conn)

	return nil
}

func (p *ConnPool) dialConn() (*Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if atomic.LoadInt32(&p.activeCount) >= int32(p.opts.PoolSize) {
		return nil, ErrPoolExhausted
	}

	conn, err := p.tryDial()
	if err != nil {
		return nil, err
	}

	atomic.AddInt32(&p.activeCount, 1)
	return &Conn{
		created:  time.Now(),
		lastUsed: time.Now(),
		grpcConn: conn,
	}, nil
}

func (p *ConnPool) tryDial() (*grpc.ClientConn, error) {
	for i := 0; i <= p.opts.MaxRetries; i++ {
		conn, err := p.opts.DialFunc()
		if err == nil {
			return conn, nil
		}

		if i == p.opts.MaxRetries {
			return nil, fmt.Errorf("connection failed after %d attempts: %w", p.opts.MaxRetries, err)
		}

		time.Sleep(time.Duration(10*(i+1)) * time.Millisecond)
	}
	return nil, ErrConnTimeout
}

func (p *ConnPool) isConnAlive(conn *Conn) bool {
	state := conn.grpcConn.GetState()
	if state == connectivity.Shutdown || state == connectivity.TransientFailure {
		return false
	}

	now := time.Now()
	if p.opts.MaxConnLifeTime > 0 && now.Sub(conn.created) > p.opts.MaxConnLifeTime {
		return false
	}

	if p.opts.MaxConnIdleTime > 0 && now.Sub(conn.lastUsed) > p.opts.MaxConnIdleTime {
		return false
	}
	return true
}

func (p *ConnPool) closeConn(conn *Conn) {
	if conn != nil && conn.grpcConn != nil {
		conn.Close()
		atomic.AddInt32(&p.activeCount, -1)
	}
}

func (p *ConnPool) isClosed() bool {
	return atomic.LoadUint32(&p.closed) == 1
}
