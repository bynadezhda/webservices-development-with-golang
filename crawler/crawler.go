package main

import (
	"fmt"
	"log"
	"sync"
)

type Fetcher interface {
	Fetch(url string) (body string, urls []string, err error)
}

type state uint8

const (
	inFlight state = iota
	done
	failed
)

type Crawler struct {
	fetcher       Fetcher
	maxGoroutines int
	sem           chan struct{}
	mu            sync.Mutex
	state         map[string]state
}

func NewCrawler(fetcher Fetcher, maxGoroutines int) *Crawler {
	return &Crawler{
		fetcher:       fetcher,
		maxGoroutines: maxGoroutines,
		sem:           make(chan struct{}, maxGoroutines),
		state:         make(map[string]state),
	}
}

func (c *Crawler) Crawl(url string, depth int, wg *sync.WaitGroup) {
	defer wg.Done()

	if depth <= 0 {
		return
	}

	c.mu.Lock()

	if s, ok := c.state[url]; ok {
		if s == inFlight || s == done {
			c.mu.Unlock()
			return
		}
	}

	c.state[url] = inFlight
	c.mu.Unlock()

	c.sem <- struct{}{}
	body, urls, err := c.fetcher.Fetch(url)
	<-c.sem

	c.mu.Lock()
	if err != nil {
		c.state[url] = failed
		c.mu.Unlock()
		log.Println(err)
		return
	}
	c.state[url] = done
	c.mu.Unlock()

	log.Printf("found: %s %q\n", url, body)

	for _, u := range urls {
		wg.Add(1)
		go func() {
			c.Crawl(u, depth-1, wg)
		}()
	}
}

func main() {
	crawler := NewCrawler(fetcher, 10)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go crawler.Crawl("https://golang.org/", 4, wg)
	wg.Wait()
}

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher is a populated fakeFetcher.
var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}
