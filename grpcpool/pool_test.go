package grpcpool

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

func TestConnPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// signalChan := make(chan os.Signal, 1)
	// signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// go func() {
	// 	<-signalChan
	// 	cancel()
	// }()

	pool, err := NewConnPool(ctx, WithDialFunc(dial))

	if err != nil {
		t.Fatalf("failed to new conn pool, err: %v\n", err)
	}
	defer pool.Close()

	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("failed to get valid conn, err: %v\n", err)
	}

	if conn.GetConn() != nil {
		t.Logf("successfully get grpc conn\n")
	}

	pool.Put(conn)

	<-ctx.Done()
}

func dial() (*grpc.ClientConn, error) {
	creds, err := credentials.NewClientTLSFromFile("./ca.crt", "")
	if err != nil {
		return nil, err
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}),
	}

	conn, err := grpc.NewClient("localhost:8080", dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}
	return conn, nil
}
