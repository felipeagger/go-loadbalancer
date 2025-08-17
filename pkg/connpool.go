package pkg

import (
	"context"
	"fmt"
	"net"
	"time"
)

func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}

func NewConnPool(addr string, size int, dialTO, idleTO time.Duration) *ConnPool {
	return &ConnPool{addr: addr, size: size, dialTO: dialTO, idleTO: idleTO}
}

func (p *ConnPool) Get(ctx context.Context) (net.Conn, error) {
	now := time.Now()
	p.mu.Lock()
	// Limpa expirados
	n := 0
	for _, pc := range p.idle {
		if now.Sub(pc.lastUsed) < p.idleTO {
			p.idle[n] = pc
			n++
		} else {
			_ = pc.Close()
			p.totalOpen--
		}
	}
	p.idle = p.idle[:n]

	// pega se houver
	if len(p.idle) > 0 {
		fmt.Printf("Reusing connection from pool: %s (idle: %d)\n", p.addr, len(p.idle))
		pc := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
		p.mu.Unlock()

		// Testa se conexão ainda está viva
		pc.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
		buf := make([]byte, 1)
		_, err := pc.Read(buf)
		pc.SetReadDeadline(time.Time{})

		if err != nil && !isTimeoutError(err) {
			fmt.Printf("Connection from pool is dead, creating new one: %s\n", p.addr)
			_ = pc.Close()
			p.mu.Lock()
			p.totalOpen--
			p.mu.Unlock()
			// Recursively call Get to create a new connection
			return p.Get(ctx)
		}

		_ = pc.SetDeadline(time.Time{}) // old = time.Now().Add(0) / clear deadline
		return pc, nil
	}
	// abre novo se caber
	if p.totalOpen < p.size {
		fmt.Println("Opening new connection with backend:", p.addr)
		p.totalOpen++
		p.mu.Unlock()
		dialer := &net.Dialer{Timeout: p.dialTO, Control: setSocketOpts}
		c, err := dialer.DialContext(ctx, "tcp", p.addr)
		if err != nil {
			p.mu.Lock()
			p.totalOpen--
			p.mu.Unlock()
			return nil, err
		}
		return c, nil
	}
	// sem slot -> dial sem contar no pool (burst)
	p.mu.Unlock()
	dialer := &net.Dialer{Timeout: p.dialTO, Control: setSocketOpts}
	return dialer.DialContext(ctx, "tcp", p.addr)
}

func (p *ConnPool) Put(c net.Conn) {
	if c == nil {
		return
	}
	pc := &pooledConn{Conn: c, lastUsed: time.Now()}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.idle) < p.size {
		p.idle = append(p.idle, pc)
	} else {
		_ = c.Close()
		p.totalOpen--
	}
}

func (p *ConnPool) MarkDead(c net.Conn) {
	fmt.Printf("Marking connection as dead: %s\n", p.addr)
	if c != nil {
		_ = c.Close()
	}
	p.mu.Lock()
	p.totalOpen-- // deixa o pool repor no futuro
	if p.totalOpen < 0 {
		p.totalOpen = 0
	}
	p.mu.Unlock()
}
