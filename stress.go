package main

import (
	"fmt"
	"net/http"
	// "crypto/tls"
	"time"
	"sync"
)

var (
	url = "http://97cm.xyz/decety/"
	goroutinesCount = 10
	correctStatusCode = 404

	count = 0
	average_time = 0
	success = 0
	mutex sync.Mutex
)



func request() bool {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return false
	}
	if resp.StatusCode != correctStatusCode {
		return false
	}
	return true
}

func worker() {
	for {
		start := time.Now()
		ok := request()
		current := time.Now().Sub(start) / time.Millisecond

		mutex.Lock()
		count += 1
		average_time += int(current)
		if ok {
			success += 1
		}
		mutex.Unlock()
	}
}

func main() {
	// insecure https
	// http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	for i := 0; i < goroutinesCount ;i++ {
		go worker()
	}
	for {
		time.Sleep(4 * time.Second)
		mutex.Lock()
		if count == 0 {
			mutex.Unlock()
			continue
		}
		fmt.Printf("Requests: %d\nSuccess: %d%%\nAverage time: %dms\n\n", count, 100 * success / count, average_time / count)
		count = 0
		average_time = 0
		success = 0
		mutex.Unlock()
	}
}