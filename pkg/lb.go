package pkg

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
)

func NewLoadBalancer(cfg *Config) *LoadBalancer {
	bes := make([]*Backend, 0, len(cfg.Backends))
	for _, a := range cfg.Backends {
		b := &Backend{Addr: a, weight: 1}
		b.Alive.Store(true) // assume vivo até provar o contrário
		bes = append(bes, b)
	}
	lb := &LoadBalancer{cfg: cfg, backends: bes}
	for _, b := range lb.backends {
		b.Pool = NewConnPool(b.Addr, cfg.PoolSizePerBackend, cfg.DialTimeout, cfg.IdleTimeout)
	}
	return lb
}

func (lb *LoadBalancer) ChooseBackend(remoteIP string) *Backend {
	alive := lb.aliveBackends()
	if len(alive) == 0 {
		fmt.Println("No alive backends")
		return nil
	}
	switch lb.cfg.LBMode {
	case "sticky":
		// sticky session by IP: consistent-ish hash (sha1 -> idx)
		h := sha1.Sum([]byte(remoteIP))
		v := binary.BigEndian.Uint32(h[:4])
		return alive[int(v)%len(alive)]
	default:
		// round-robin
		i := int(lb.rrIdx.Add(1)-1) % len(alive)
		return alive[i]
	}
}

func (lb *LoadBalancer) aliveBackends() []*Backend {
	out := make([]*Backend, 0, len(lb.backends))
	for _, b := range lb.backends {
		if b.Alive.Load() {
			out = append(out, b)
		}
	}
	return out
}
