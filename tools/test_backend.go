package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	port := "5001"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	hostname, _ := os.Hostname()
	pid := os.Getpid()

	fmt.Printf("TCP server started on port %s\n", port)
	fmt.Printf("HOST: %s | PID: %d\n", hostname, pid)
	fmt.Println("Press Ctrl+C to stop")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(5 * time.Minute)
		}

		go handleConnection(conn, hostname, pid)
	}
}

func handleConnection(conn net.Conn, hostname string, pid int) {
	is_health_check := false
	defer func() {
		conn.Close()
		if !is_health_check {
			fmt.Println("Connection closed")
		}
	}()

	buf := make([]byte, 1024)

	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		if n == 0 {
			return
		}

		if strings.Contains(string(buf[:n]), "ping") {
			is_health_check = true
			conn.Write([]byte("pong\r\n\r\n"))
			return
		}

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		response := fmt.Sprintf("HOST: %s | PID: %d | TIME: %s\n", hostname, pid, timestamp)

		conn.SetWriteDeadline(time.Now().Add(5 * time.Minute))
		conn.Write([]byte(response))
	}
}
