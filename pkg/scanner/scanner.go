package scanner

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type Result struct {
	Host   string
	Port   int
	Result bool
	Banner string
}

func ScanPort(ctx context.Context, host string, port int, timeout time.Duration) Result {
	address := fmt.Sprintf("%s:%d", host, port)
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp4", address)
	if err != nil {
		return Result{Host: host, Port: port, Result: false}
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

	return Result{Host: host, Port: port, Result: true, Banner: banner}
}

func Hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var hosts []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		hosts = append(hosts, ip.String())
	}

	if len(hosts) > 2 {
		return hosts[1 : len(hosts)-1], nil
	}
	return hosts, nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
