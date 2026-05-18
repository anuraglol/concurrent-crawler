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

		title, links, err := Crawl(normalized)
		if err != nil {
			fmt.Println("error:", err)
			wg.Done()
			continue
		}

		results <- Result{
			Title: title,
			URL:   normalized,
			Links: links,
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
	results := make(chan Result, maxQueue)

	visited := make(map[string]bool)

	var allResults []Result
	var resultsWg sync.WaitGroup
	resultsWg.Add(1)
	go func() {
		defer resultsWg.Done()
		for r := range results {
			allResults = append(allResults, r)
		}
	}()

	for i := 0; i < maxWorkers; i++ {
		go worker(
			i,
			jobs,
			visited,
			&mu,
			&wg,
			results,
		)
	}

	wg.Add(1)

	jobs <- Job{
		URL:   baseURL,
		Depth: maxDepth,
	}

	wg.Wait()
	close(results)
	resultsWg.Wait()

	fmt.Printf("crawled %d pages\n", len(allResults))
	fmt.Printf("\nfinished in %v\n", time.Since(start))

	for _, tmp := range allResults {
		fmt.Printf("\nfound, title: %v, url: %v\n", tmp.Title, tmp.URL)
	}
}
