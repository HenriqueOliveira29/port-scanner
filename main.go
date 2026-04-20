package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

type Result struct {
	port   int
	result bool
	banner string
}

func ScanPort(ctx context.Context, host string, port int, timeout time.Duration) Result {
    address := fmt.Sprintf("%s:%d", host, port)
    var d net.Dialer
    conn, err := d.DialContext(ctx, "tcp4", address)
    if err != nil {
        return Result{port: port, result: false}
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

    return Result{port: port, result: true, banner: banner}
}

func worker(ctx context.Context, wg *sync.WaitGroup, host string, ports chan int, result chan Result) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case port, ok := <-ports:
			if !ok {
				return
			}
			result <- ScanPort(ctx, host, port, 2*time.Second)
		}
	}
}

func workingPool(ctx context.Context, numberOfWorkers int, host string, ports chan int, result chan Result) {
	var wg sync.WaitGroup
	for i := 0; i < numberOfWorkers; i++ {
		wg.Add(1)
		go worker(ctx, &wg, host, ports, result)
	}
	wg.Wait()
}

func allocate(ctx context.Context, NumberOfPorts int, ports chan<- int) {
	defer close(ports)
	for i := 0; i < NumberOfPorts; i++ {
		select {
		case <-ctx.Done():
			return
		case ports <- i:
		}
	}
}

func showResult(result chan Result) {
	for res := range result {
		if res.result {
			if res.banner != "" {
				fmt.Printf("[+] Port %d is open | Banner: %s\n", res.port, res.banner)
			} else {
				fmt.Printf("[+] Port %d is open | (No banner)\n", res.port)
			}
		}
	}
}

func main() {
	target := flag.String("t", "127.0.0.1", "Alvo para escanear")
	workers := flag.Int("w", 100, "Número de workers simultâneos")
	nports := flag.Int("p", 100, "Número de ports para escanear")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	ports := make(chan int, 100)
	result := make(chan Result, 100)

	starttime := time.Now()

	go allocate(ctx, *nports, ports)

	go func() {
		workingPool(ctx, *workers, *target, ports, result)
		close(result)
	}()

	showResult(result)

	endtime := time.Now()
	diff := endtime.Sub(starttime)

	fmt.Printf("Varredura de %d portas concluída em %f segundos\n", *nports, diff.Seconds())
}
