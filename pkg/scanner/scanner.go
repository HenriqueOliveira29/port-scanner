package scanner

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type Result struct {
	Port   int
	Result bool
	Banner string
}

func ScanPort(ctx context.Context, host string, port int, timeout time.Duration) Result {
	address := fmt.Sprintf("%s:%d", host, port)
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp4", address)
	if err != nil {
		return Result{Port: port, Result: false}
	}
	defer conn.Close()

	if port == 80 || port == 443 || port == 8080 {
		conn.SetWriteDeadline(time.Now().Add(timeout))
		conn.Write([]byte("HEAD / HTTP/1.0\r\n\r\n"))
	}

	conn.SetReadDeadline(time.Now().Add(timeout))

	buffer := make([]byte, 512)
	n, err := conn.Read(buffer)

	var banner string
	if err == nil && n > 0 {
		raw := string(buffer[:n])
		lines := strings.Split(raw, "\n")
		banner = strings.TrimSpace(lines[0])
	}

	return Result{Port: port, Result: true, Banner: banner}
}
