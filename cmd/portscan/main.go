package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"example/portScanner/pkg/scanner"
)

func worker(ctx context.Context, wg *sync.WaitGroup, host string, ports chan int, result chan scanner.Result) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case port, ok := <-ports:
			if !ok {
				return
			}
			result <- scanner.ScanPort(ctx, host, port, 2*time.Second)
		}
	}
}

func workingPool(ctx context.Context, numberOfWorkers int, host string, ports chan int, result chan scanner.Result) {
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

func showResult(result chan scanner.Result, nports *int) {
	count := 0
	for res := range result {
		count++
		percent := (float64(count) / float64(*nports)) * 100

		bar := strings.Repeat("=", int(percent/5)) + strings.Repeat("-", 20-int(percent/5))
		fmt.Printf("\r[%s] %.1f%% (%d/%d)", bar, percent, count, *nports)

		if res.Result {
			fmt.Printf("\r[+] Porta %d aberta | %s                                \n", res.Port, res.Banner)
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
	result := make(chan scanner.Result, 100)

	starttime := time.Now()

	go allocate(ctx, *nports, ports)

	go func() {
		workingPool(ctx, *workers, *target, ports, result)
		close(result)
	}()

	showResult(result, nports)

	endtime := time.Now()
	diff := endtime.Sub(starttime)

	fmt.Printf("Varredura de %d portas concluída em %f segundos\n", *nports, diff.Seconds())
}
