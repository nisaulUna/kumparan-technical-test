package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	const totalRequests = 100
	const concurrency = 20

	url := "http://localhost:8080/articles?limit=20"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var wg sync.WaitGroup
	var success atomic.Int64
	var failed atomic.Int64

	sem := make(chan struct{}, concurrency)
	start := time.Now()

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			resp, err := client.Get(url)
			if err != nil {
				failed.Add(1)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				success.Add(1)
			} else {
				failed.Add(1)
			}
		}()
	}

	wg.Wait()

	duration := time.Since(start)

	fmt.Println("Total requests:", totalRequests)
	fmt.Println("Concurrent requests:", concurrency)
	fmt.Println("Successful:", success.Load())
	fmt.Println("Failed:", failed.Load())
	fmt.Println("Total duration:", duration)
	fmt.Printf(
		"Throughput: %.2f req/s\n",
		float64(totalRequests)/duration.Seconds(),
	)
}
