package grpcpool

type Stats struct {
	// number of times free connection was found in the pool
	HitCount uint32

	// number of times free connection was NOT found in the pool
	MissCount uint32

	// number of times a wait timeout occurred
	TimeoutCount uint32

	// number of times a connection was waited
	WaitCount uint32

	// number of total connections in the pool
	TotalConns uint32

	// number of idle connections in the pool
	IdleConns uint32

	// number of stale connections removed from the pool
	StaleConns uint32
}
