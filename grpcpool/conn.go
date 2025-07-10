package grpcpool

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type Conn struct {
	created  time.Time
	lastUsed time.Time
	grpcConn *grpc.ClientConn

	onClose func() error
}

func NewConn(grpcConn *grpc.ClientConn) *Conn {
	return &Conn{
		grpcConn: grpcConn,
	}
}

func (c *Conn) Close() error {
	if c.onClose != nil {
		c.onClose()
	}
	return c.grpcConn.Close()
}

func (c *Conn) GetState() connectivity.State {
	return c.grpcConn.GetState()
}

func (c *Conn) GetConn() *grpc.ClientConn {
	return c.grpcConn
}
