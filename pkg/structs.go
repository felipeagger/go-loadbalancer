package pkg

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	Addr   string
	Alive  atomic.Bool
	Pool   *ConnPool
	weight int
}

type Config struct {
	ListenAddr         string
	Backends           []string
	LBMode             string // "rr" | "sticky"
	EnableProxyProtoV1 bool
	DialTimeout        time.Duration
	IdleTimeout        time.Duration
	IOTimeout          time.Duration
	PoolSizePerBackend int
	HealthInterval     time.Duration
	ReadBufSize        int
	WriteBufSize       int
}

type LoadBalancer struct {
	cfg      *Config
	backends []*Backend
	rrIdx    atomic.Uint32
}

type pooledConn struct {
	net.Conn
	lastUsed time.Time
}

type ConnPool struct {
	addr       string
	size       int
	dialTO     time.Duration
	idleTO     time.Duration
	mu         sync.Mutex
	idle       []*pooledConn
	totalOpen  int
	lastHealth atomic.Int64
}
