package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	baseURL    = "https://anurag.tech"
	maxWorkers = 5
	maxQueue   = 10000
	maxDepth   = 2
)

type Job struct {
	URL   string
	Depth int
}

type Result struct {
	Title string
	URL   string
	Links []string
}

func worker(
	id int,
	jobs chan Job,
	visited map[string]bool,
	mu *sync.Mutex,
	wg *sync.WaitGroup,
	results chan<- Result,
	ctx context.Context, // Context should be passed directly, not as a pointer
) {
	for {
		select {
		case <-ctx.Done():
			return // Exit worker cleanly if context is cancelled
		case job, ok := <-jobs:
			if !ok {
				return // Channel closed, time to exit
			}

			if job.Depth <= 0 {
				wg.Done()
				continue
			}

			normalized, err := normalizeURL(job.URL)
			if err != nil {
				wg.Done()
				continue
			}

			if !sameHost(baseURL, normalized) {
				wg.Done()
				continue
			}

			mu.Lock()
			if visited[normalized] {
				mu.Unlock()
				wg.Done()
				continue
			}
			visited[normalized] = true
			mu.Unlock()

			fmt.Printf("[worker %d] crawling: %s\n", id, normalized)

			title, links, err := Crawl(normalized)
			if err != nil {
				fmt.Println("error:", err)
				wg.Done()
				continue
			}

			select {
			case <-ctx.Done():
				wg.Done()
				return
			case results <- Result{Title: title, URL: normalized, Links: links}:
			}

			for _, link := range links {
				wg.Add(1)
				go func(l string, d int) {
					select {
					case <-ctx.Done():
						wg.Done() // Drop job if shutting down
					case jobs <- Job{URL: l, Depth: d}:
					}
				}(link, job.Depth-1)
			}

			wg.Done()
		}
	}
}

func main() {
	start := time.Now()

	var wg sync.WaitGroup
	var mu sync.Mutex

	jobs := make(chan Job, maxQueue)
	results := make(chan Result, maxQueue)
	visited := make(map[string]bool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var allResults []Result
	var resultsWg sync.WaitGroup
	resultsWg.Add(1)
	go func() {
		defer resultsWg.Done()
		for r := range results {
			allResults = append(allResults, r)
		}
	}()

	for i := range maxWorkers {
		go worker(i, jobs, visited, &mu, &wg, results, ctx)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var stopped int32 // Thread-safe flag to ensure we only print shutdown messages once
	shutdownTriggered := func(reason string) {
		if atomic.CompareAndSwapInt32(&stopped, 0, 1) {
			fmt.Printf("\n[%s] Initiating shutdown, clearing queues...\n", reason)
			cancel()
			go func() {
				for range jobs {
				}
			}()
		}
	}

	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			input, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(input)) == "q" {
				shutdownTriggered("'q' keypress")
				return
			}
		}
	}()

	go func() {
		<-sigChan
		shutdownTriggered("Ctrl+C")
	}()

	wg.Add(1)
	jobs <- Job{
		URL:   baseURL,
		Depth: maxDepth,
	}

	wg.Wait()
	close(jobs)
	close(results)
	resultsWg.Wait()

	fmt.Printf("\ncrawled %d pages\n", len(allResults))
	fmt.Printf("finished in %v\n", time.Since(start))

	for _, tmp := range allResults {
		fmt.Printf("found, title: %v, url: %v\n", tmp.Title, tmp.URL)
	}
}
