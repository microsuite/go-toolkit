package grpcpool

import (
	"time"

	"google.golang.org/grpc"
)

type Options struct {
	// DialFunc is the function to create a new connection.
	DialFunc func() (*grpc.ClientConn, error)

	// PoolSize is the size of connection pool
	PoolSize int

	// MaxRetries is the maximum number of retries before giving up.
	// -1 (not 0) disables retries.
	//
	// default: 3 retries
	MaxRetries int

	// MinIdleConns is the minimum number of idle connections in the pool.
	MinIdleConns int

	// MaxIdleConns is the maximum number of idle connections in the pool.
	MaxIdleConns int

	// MaxActiveConns is the maximum number of connections allowed in the pool.
	MaxActiveConns int

	// MaxConnIdleTime is the maximum amount of time a connection may be idle before
	// being closed.
	MaxConnIdleTime time.Duration

	// MaxConnLifeTime is the maximum amount of time a connection may be reused.
	MaxConnLifeTime time.Duration

	// IdleCheckFreq is the frequency of checking the connection idle.
	IdleCheckFreq time.Duration

	// PoolTimeout is the amount of time client waits for connection if all connections
	// are busy before returning an error.
	PoolTimeout time.Duration
}

type Option func(*Options)

func NewOptions(opts ...Option) *Options {
	o := &Options{
		PoolSize:        10,
		MaxRetries:      3,
		MinIdleConns:    2,
		MaxIdleConns:    5,
		MaxActiveConns:  20,
		MaxConnIdleTime: 3 * time.Minute,
		MaxConnLifeTime: 24 * time.Hour,
		PoolTimeout:     1 * time.Second,
		IdleCheckFreq:   time.Minute,
	}

	for _, opt := range opts {
		opt(o)
	}
	return o
}

func WithDialFunc(fn func() (*grpc.ClientConn, error)) Option {
	return func(o *Options) {
		o.DialFunc = fn
	}
}

func WithPoolSize(size int) Option {
	return func(o *Options) {
		o.PoolSize = size
	}
}

func WithMaxRetries(max int) Option {
	return func(o *Options) {
		o.MaxRetries = max
	}
}

func WithMinIdleConns(min int) Option {
	return func(o *Options) {
		o.MinIdleConns = min
	}
}

func WithMaxIdleConns(max int) Option {
	return func(o *Options) {
		o.MaxIdleConns = max
	}
}

func WithMaxActiveConns(max int) Option {
	return func(o *Options) {
		o.MaxActiveConns = max
	}
}

func WithMaxConnIdleTime(t time.Duration) Option {
	return func(o *Options) {
		o.MaxConnIdleTime = t
	}
}

func WithMaxConnLifeTime(t time.Duration) Option {
	return func(o *Options) {
		o.MaxConnLifeTime = t
	}
}

func WithPoolTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.PoolTimeout = t
	}
}

func WithIdleCheckFreq(t time.Duration) Option {
	return func(o *Options) {
		o.IdleCheckFreq = t
	}
}
