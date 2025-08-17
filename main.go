package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/felipeagger/go-loadbalancer/pkg"
)

func main() {
	cfg := &pkg.Config{
		ListenAddr:         pkg.EnvStr("LISTEN", ":4000"),
		Backends:           strings.Split(pkg.EnvStr("BACKENDS", "127.0.0.1:5001,127.0.0.1:5002"), ","),
		LBMode:             pkg.EnvStr("LB_MODE", "rr"), // "rr" ou "sticky"
		EnableProxyProtoV1: pkg.EnvBool("PROXY_V1", false),
		DialTimeout:        pkg.EnvDur("DIAL_TIMEOUT", 1000*time.Millisecond),
		IdleTimeout:        pkg.EnvDur("IDLE_TIMEOUT", 3*time.Minute),
		IOTimeout:          pkg.EnvDur("IO_TIMEOUT", 0), // 0 = sem deadline
		PoolSizePerBackend: pkg.EnvInt("POOL_SIZE", 10), // 256
		HealthInterval:     pkg.EnvDur("HEALTH_EVERY", 5*time.Second),
		ReadBufSize:        64 << 10,
		WriteBufSize:       64 << 10,
	}

	lb := pkg.NewLoadBalancer(cfg)

	go pkg.HealthLoop(lb, cfg.HealthInterval)

	lcfg := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var serr error
			if err := c.Control(func(fd uintptr) {
				serr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if serr != nil {
					return
				}
				serr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 0x0F, 1) // SO_REUSEPORT = 15 on Linux
			}); err != nil {
				return err
			}
			return serr
		},
	}

	ln, err := lcfg.Listen(context.Background(), "tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("listening on %s -> backends=%v mode=%s", cfg.ListenAddr, cfg.Backends, cfg.LBMode)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		c, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				break
			}
			log.Printf("accept err: %v", err)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleConn(ctx, c, lb, cfg)
		}()
	}

	//wg.Wait() // TODO: adjust to timeout 30s and close connections
	log.Println("shutdown complete")
}

func handleConn(ctx context.Context, client net.Conn, lb *pkg.LoadBalancer, cfg *pkg.Config) {
	defer client.Close()

	remoteIP := pkg.ExtractIP(client.RemoteAddr())
	be := lb.ChooseBackend(remoteIP)
	if be == nil {
		log.Printf("no alive backends for %s", remoteIP)
		return
	}

	log.Printf("Chosen backend: %s for client: %s", be.Addr, remoteIP)
	back, err := be.Pool.Get(ctx)
	if err != nil {
		log.Printf("dial backend %s: %v", be.Addr, err)
		be.Alive.Store(false)
		return
	}

	// (Opcional) Envia cabeçalho PROXY v1
	if cfg.EnableProxyProtoV1 {
		h := fmt.Sprintf("PROXY TCP4 %s 0 0 0\r\n", remoteIP)
		_ = back.SetWriteDeadline(pkg.Deadline(cfg.IOTimeout))
		if _, err := io.WriteString(back, h); err != nil {
			be.Pool.MarkDead(back)
			return
		}
		_ = back.SetWriteDeadline(time.Time{})
	}

	// deadlines (se configuradas)
	if cfg.IOTimeout > 0 {
		_ = client.SetDeadline(time.Now().Add(cfg.IOTimeout))
		_ = back.SetDeadline(time.Now().Add(cfg.IOTimeout))
	}

	// pipeline bidirecional
	errc := make(chan error, 2)

	go proxyOneWay(client, back, errc)
	go proxyOneWay(back, client, errc)

	// espera um dos lados encerrar
	err1 := <-errc
	// remove deadlines
	if cfg.IOTimeout > 0 {
		_ = client.SetDeadline(time.Time{})
		_ = back.SetDeadline(time.Time{})
	}

	// encerra outro lado também
	_ = halfClose(client)
	//_ = halfClose(back) // try keep connection open

	// drena segundo erro
	select {
	case <-errc:
	default:
	}

	// decide se devolve conexão ao pool
	if err1 == nil {
		be.Pool.Put(back)
	} else {
		log.Printf("Connection failed (%v), marking as dead: %s", err1, be.Addr)
		be.Pool.MarkDead(back)
	}
}

func proxyOneWay(src net.Conn, dst net.Conn, errc chan<- error) {
	_, err := io.Copy(dst, src)
	errc <- err
}

func halfClose(c net.Conn) error {
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
		_ = tc.CloseRead()
		return tc.Close()
	}
	return c.Close()
}
