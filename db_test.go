package main

import (
	"testing"
	"fmt"
	"sync"
	"strconv"
	"strings"
	"math/rand"
	"encoding/json"
)

var (
	max_goroutines = 1
	productCount = 100
	colorCount = 3
	typeCount = 50
	photoPerTypeCount = 18

	photoCount = 1000
	image_ids = make ([]string, photoCount)
	wg = sync.WaitGroup{}
)

func benchUploading(t *testing.T) {
	sem := make(chan int, max_goroutines)
	for i := 0; i < photoCount; i++ {
		sem <- 1
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			status_code, body := uploadPhoto(t, "test_token", "./mini-sample.jpg")
			if status_code != 200 {
				t.Fatalf("Status code doesn't equal 200")
			}

			var value map[string]interface{}
			if err := json.Unmarshal([]byte(body), &value); err != nil {
				t.Fatalf("Error parsing json: %v", err)
			}

			if value["error"].(string) != "" {
				t.Fatalf("Failed uploading photo: %v", value["error"].(string))
			}
			if value["result"].(string) == "" {
				t.Fatalf("Failed uploading photo")
			}

			image_ids[i] = value["result"].(string)
			
			<- sem
		}(i)
	}
	wg.Wait()
}

func benchUpdating(t *testing.T) {
	sem := make(chan int, max_goroutines)
	for i := 0; i < productCount; i++ {
		for j := 0; j < colorCount; j++ {
			for h := 0; h < typeCount; h++ {
				sem <- 1
				wg.Add(1)
				go func(i,j,h int) {
					defer wg.Done()

					ids := make([]string, photoPerTypeCount)
					for i := range ids {
						ids[i] = image_ids[rand.Intn(photoCount)]
					}

					data := map[string]string{
						"token": "test_token",
						"id": strconv.Itoa(i) + "qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq",
						"color": strconv.Itoa(j),
						"type": strconv.Itoa(h),
						"size": "M",
						"d1": fmt.Sprint(rand.Float64()),
						"d2": fmt.Sprint(rand.Float64()),
						"d3": fmt.Sprint(rand.Float64()),
						"d4": fmt.Sprint(rand.Float64()),
						"d5": fmt.Sprint(rand.Float64()),
						"image_ids": strings.Join(ids, ","),
					}
					resp, body := request(baseURL + "update", "POST", data, map[string]string{})

					if resp.StatusCode != 200 || string(body) != `{"error":"","result":""}` {
						t.Fatalf("Failed updating data")
					}

					<- sem
				}(i,j,h)
			}
		}
	}
	wg.Wait()
}

