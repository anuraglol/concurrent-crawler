package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/html"
)

func visit(n *html.Node, baseURL *url.URL, links []string) []string {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				u, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}

				resolved := baseURL.ResolveReference(u)

				links = append(links, resolved.String())
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		links = visit(c, baseURL, links)
	}

	return links
}

func Crawl(rawURL string) ([]string, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	var links []string

	links = visit(doc, baseURL, links)

	return links, nil
}

func main() {
	start := time.Now()
	var wg sync.WaitGroup
	linksCh := make(chan string)

	urls := []string{
		"https://anurag.tech",
	}

	for _, url := range urls {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			links, err := Crawl(url)
			if err != nil {
				fmt.Println("error:", err)
			} else {
				for _, link := range links {
					linksCh <- link
				}
			}
		}(url)
	}

	go func() {
		wg.Wait()
		close(linksCh)
	}()

	for link := range linksCh {
		fmt.Println(link)
	}

	fmt.Printf("program finished in: %v\n", time.Since(start))
}
