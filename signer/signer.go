package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

const hashCount = 6

var md5Lock sync.Mutex

func ExecutePipeline(jobs ...job) {
	var in chan interface{}
	var wg sync.WaitGroup

	for _, j := range jobs {
		out := make(chan interface{}, 10)

		wg.Add(1)
		go func(j job, in, out chan interface{}) {
			defer wg.Done()
			j(in, out)
			close(out)
		}(j, in, out)

		in = out
	}

	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup

	for raw := range in {
		val, ok := raw.(int)
		if !ok {
			continue
		}
		data := strconv.Itoa(val)

		wg.Add(1)
		go func(data string) {
			defer wg.Done()

			md5chan := make(chan string, 1)
			crc32chan := make(chan string, 1)
			crc32md5chan := make(chan string, 1)

			go func() {
				defer close(md5chan)
				md5Lock.Lock()
				md5 := DataSignerMd5(data)
				md5Lock.Unlock()
				md5chan <- md5
			}()

			go func() {
				defer close(crc32chan)
				crc32chan <- DataSignerCrc32(data)
			}()

			go func() {
				defer close(crc32md5chan)
				md5 := <-md5chan
				crc32md5chan <- DataSignerCrc32(md5)
			}()

			crc32 := <-crc32chan
			crc32md5 := <-crc32md5chan
			out <- crc32 + "~" + crc32md5
		}(data)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	var wg sync.WaitGroup

	for v := range in {
		wg.Add(1)
		go func(val interface{}) {
			defer wg.Done()
			s, ok := val.(string)
			if !ok {
				return
			}
			calculateCrc32Th(out, s)
		}(v)
	}

	wg.Wait()
}

func calculateCrc32Th(out chan interface{}, data string) {
	var wg sync.WaitGroup
	hashes := make([]string, hashCount)

	for i := 0; i < hashCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			hashes[i] = DataSignerCrc32(strconv.Itoa(i) + data)
		}()
	}

	wg.Wait()

	out <- strings.Join(hashes, "")
}

func CombineResults(in, out chan interface{}) {
	var results []string

	for v := range in {
		s, ok := v.(string)
		if !ok {
			continue
		}
		results = append(results, s)
	}

	sort.Strings(results)

	out <- strings.Join(results, "_")
}
