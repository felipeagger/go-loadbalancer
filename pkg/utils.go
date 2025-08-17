package pkg

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
)

func EnvStr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func EnvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		var x int
		_, _ = fmt.Sscanf(v, "%d", &x)
		if x > 0 {
			return x
		}
	}
	return def
}

func EnvDur(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return def
}

func EnvBool(k string, def bool) bool {
	if v := os.Getenv(k); v != "" {
		v2 := strings.ToLower(strings.TrimSpace(v))
		return v2 == "1" || v2 == "true" || v2 == "yes" || v2 == "on"
	}
	return def
}

func setSocketOpts(network, address string, c syscall.RawConn) error {
	var serr error
	if err := c.Control(func(fd uintptr) {
		// TCP_NODELAY (desabilita Nagle) melhora latência de mensagens pequenas
		serr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1)
		if serr != nil {
			return
		}
		// aumentando buffers pode melhorar throughput sob RTT alto
		_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 1<<20) // 1 MiB
		_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 1<<20)
	}); err != nil {
		return err
	}
	return serr
}

func ExtractIP(addr net.Addr) string {
	h, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return h
}

func Deadline(d time.Duration) time.Time {
	if d <= 0 {
		return time.Time{}
	}
	return time.Now().Add(d)
}
