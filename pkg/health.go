package pkg

import (
	"fmt"
	"math/rand/v2"
	"net"
	"time"
)

func HealthLoop(lb *LoadBalancer, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for range t.C {
		for _, b := range lb.backends {
			addr := b.Addr
			go func(be *Backend, addr string) {
				d := &net.Dialer{Timeout: 2 * time.Second, Control: setSocketOpts}
				c, err := d.Dial("tcp", addr)
				if err != nil {
					fmt.Println("Error health check backend:", err)
					be.Alive.Store(false)
					return
				}
				c.Write([]byte("ping\r\n\r\n"))
				_ = c.Close()
				be.Alive.Store(true)
			}(b, addr)

			// “poking” no pool pra reciclar conexões muito antigas (randomly)
			if b.Pool != nil && rand.IntN(64) == 0 {
				b.Pool.mu.Lock()
				for len(b.Pool.idle) > 0 && time.Since(b.Pool.idle[0].lastUsed) > b.Pool.idleTO {
					pc := b.Pool.idle[0]
					b.Pool.idle = b.Pool.idle[1:]
					_ = pc.Close()
					b.Pool.totalOpen--
				}
				b.Pool.mu.Unlock()
			}
		}
	}
}
