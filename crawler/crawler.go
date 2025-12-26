package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

var maxGoroutines = 10
var maxRetries = 5

type Fetcher interface {
	Fetch(ctx context.Context, url string) (body string, urls []string, err error)
}

type stateType uint8

const (
	stateInFlight stateType = iota
	stateDone
	stateFailed
)

type urlState struct {
	status  stateType
	retries int
}

type Crawler struct {
	fetcher       Fetcher
	maxGoroutines int
	maxRetries    int
	sem           chan struct{}
	mu            sync.Mutex
	state         map[string]*urlState
}

func NewCrawler(fetcher Fetcher, maxGoroutines int) *Crawler {
	return &Crawler{
		fetcher:       fetcher,
		maxGoroutines: maxGoroutines,
		maxRetries:    maxRetries,
		sem:           make(chan struct{}, maxGoroutines),
		state:         make(map[string]*urlState),
	}
}

func (c *Crawler) Crawl(ctx context.Context, url string, depth int, wg *sync.WaitGroup) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		return
	default:
	}

	if depth <= 0 {
		return
	}

	c.mu.Lock()

	if s, ok := c.state[url]; ok {
		if s.status == stateInFlight || s.status == stateDone {
			c.mu.Unlock()
			return
		}

		if s.status == stateFailed && s.retries >= c.maxRetries {
			c.mu.Unlock()
			return
		}
	}

	if c.state[url] == nil {
		c.state[url] = &urlState{status: stateInFlight, retries: 0}
	} else {
		c.state[url].status = stateInFlight
		c.state[url].retries++
	}

	c.mu.Unlock()

	select {
	case c.sem <- struct{}{}:
	case <-ctx.Done():
		c.mu.Lock()
		c.state[url].status = stateFailed
		c.mu.Unlock()
		return
	}

	body, urls, err := c.fetcher.Fetch(ctx, url)
	<-c.sem

	c.mu.Lock()
	if err != nil {
		c.state[url].status = stateFailed
		c.mu.Unlock()

		if ctx.Err() == nil {
			log.Println(err)
		}
		return
	}
	c.state[url].status = stateDone
	c.mu.Unlock()

	log.Printf("found: %s %q\n", url, body)

	for _, u := range urls {
		select {
		case <-ctx.Done():
			return
		default:
		}

		wg.Add(1)
		go c.Crawl(ctx, u, depth-1, wg)
	}
}

func main() {
	crawler := NewCrawler(fetcher, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go crawler.Crawl(ctx, "https://golang.org/", 4, wg)
	wg.Wait()
}

type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(ctx context.Context, url string) (string, []string, error) {
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

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
