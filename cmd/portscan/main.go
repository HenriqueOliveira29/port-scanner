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

type Task struct {
	Host string
	Port int
}

func worker(ctx context.Context, wg *sync.WaitGroup, tasks chan Task, result chan scanner.Result) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			result <- scanner.ScanPort(ctx, task.Host, task.Port, 1*time.Second)
		}
	}
}

func workingPool(ctx context.Context, numberOfWorkers int, tasks chan Task, result chan scanner.Result) {
	var wg sync.WaitGroup
	for i := 0; i < numberOfWorkers; i++ {
		wg.Add(1)
		go worker(ctx, &wg, tasks, result)
	}
	wg.Wait()
}

func allocate(ctx context.Context, nports *int, hosts []string, tasks chan Task) {
	defer close(tasks)
	for _, h := range hosts {
		for i := 1; i <= *nports; i++ {
			select {
			case <-ctx.Done():
				return
			case tasks <- Task{Host: h, Port: i}:
			}
		}
	}
}

func showResult(result chan scanner.Result, totalTasks int) {
	count := 0
	for res := range result {
		count++
		percent := (float64(count) / float64(totalTasks)) * 100

		done := int(percent / 5)
		if done > 20 {
			done = 20
		}
		todo := 20 - done
		if todo < 0 {
			todo = 0
		}

		bar := strings.Repeat("=", done) + strings.Repeat("-", todo)
		fmt.Printf("\r[%s] %.1f%% (%d/%d)", bar, percent, count, totalTasks)

		if res.Result {
			fmt.Printf("\r[+] Porta %d aberta no IP %s | %s                                \n", res.Port, res.Host, res.Banner)
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

	tasks := make(chan Task, 1000)
	hosts, _ := scanner.Hosts(*target)
	result := make(chan scanner.Result, 100)

	starttime := time.Now()

	go allocate(ctx, nports, hosts, tasks)

	go func() {
		workingPool(ctx, *workers, tasks, result)
		close(result)
	}()

	showResult(result, len(hosts)*(*nports))

	endtime := time.Now()
	diff := endtime.Sub(starttime)

	fmt.Printf("Varredura de %d portas concluída em %f segundos\n", *nports, diff.Seconds())
}
