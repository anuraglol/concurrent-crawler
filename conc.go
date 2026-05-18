package main

import (
	"fmt"
	"sync"
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

func worker(
	id int,
	jobs chan Job,
	visited map[string]bool,
	mu *sync.Mutex,
	wg *sync.WaitGroup,
) {
	for job := range jobs {
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

		links, err := Crawl(normalized)
		if err != nil {
			fmt.Println("error:", err)
			wg.Done()
			continue
		}

		for _, link := range links {
			fmt.Println("found:", link)

			wg.Add(1)

			jobs <- Job{
				URL:   link,
				Depth: job.Depth - 1,
			}
		}

		wg.Done()
	}
}

func main() {
	start := time.Now()

	var wg sync.WaitGroup
	var mu sync.Mutex

	jobs := make(chan Job, maxQueue)

	visited := make(map[string]bool)

	for i := 0; i < maxWorkers; i++ {
		go worker(
			i,
			jobs,
			visited,
			&mu,
			&wg,
		)
	}

	wg.Add(1)

	jobs <- Job{
		URL:   baseURL,
		Depth: maxDepth,
	}

	wg.Wait()

	fmt.Printf("\nfinished in %v\n", time.Since(start))
}
