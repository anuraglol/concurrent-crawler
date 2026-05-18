package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func sameHost(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}

	ub, err := url.Parse(b)
	if err != nil {
		return false
	}

	return ua.Host == ub.Host
}

func normalizeURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	if strings.HasSuffix(u.Path, "/") && u.Path != "/" {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}

	u.Fragment = ""

	q := u.Query()
	u.RawQuery = q.Encode()

	return u.String(), nil
}

var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: time.Second * 10,
}

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

func Crawl(rawURL string) (string, []string, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", nil, err
	}

	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return "", nil, err
	}

	var links []string
	links = visit(doc, baseURL, links)
	title := extractTitle(doc)

	return title, links, nil
}

func extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		if n.FirstChild != nil {
			return strings.TrimSpace(n.FirstChild.Data)
		}
		return ""
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := extractTitle(c); title != "" {
			return title
		}
	}
	return ""
}
