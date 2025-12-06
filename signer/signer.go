package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

var md5Lock = &sync.Mutex{}

func ExecutePipeline(jobs ...job) {
	var in chan interface{} = nil
	wg := &sync.WaitGroup{}

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
	wg := &sync.WaitGroup{}

	for raw := range in {
		data := strconv.Itoa((raw.(int)))
		wg.Add(1)

		go func(raw interface{}, out chan interface{}) {
			defer wg.Done()

			md5chan := make(chan string)
			crc32chan := make(chan string)
			crc32md5chan := make(chan string)

			go func() {
				md5Lock.Lock()
				md5 := DataSignerMd5(data)
				md5Lock.Unlock()
				md5chan <- md5
			}()

			go func() {
				crc32chan <- DataSignerCrc32(data)
			}()

			go func() {
				md5 := <-md5chan
				crc32md5chan <- DataSignerCrc32(md5)
			}()

			out <- <-crc32chan + "~" + <-crc32md5chan
		}(raw, out)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for v := range in {
		wg.Add(1)
		go func(out chan interface{}, v interface{}) {
			defer wg.Done()
			calculateCrc32Th(out, v.(string))
		}(out, v)
	}

	wg.Wait()
}

func calculateCrc32Th(out chan interface{}, data string) {
	var wg sync.WaitGroup
	hashes := make([]string, 6)

	for i := 0; i < 6; i++ {
		wg.Add(1)
		iter := i

		go func() {
			defer wg.Done()
			hashes[iter] = DataSignerCrc32(strconv.Itoa(iter) + data)
		}()
	}

	wg.Wait()

	var b strings.Builder
	for i := 0; i < 6; i++ {
		b.WriteString(hashes[i])
	}

	out <- b.String()
}

func CombineResults(in, out chan interface{}) {
	var results []string

	for v := range in {
		results = append(results, v.(string))
	}

	sort.Strings(results)

	out <- strings.Join(results, "_")
}
