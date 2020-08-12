package main

import (
	"time"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"os"
	"io"
	"mime/multipart"
	"bytes"
	"encoding/json"
	"context"
	"strings"
	"strconv"
)

var (
	baseURL = "http://localhost:32851/decety/"

	imageIDsByToken = map[string][]string{}
)

func request(URL, method string, data, cookies map[string]string) (*http.Response, []byte) {
	buffer := new(bytes.Buffer)
	params := url.Values{}
	for key, value := range data {
		params.Set(key, value)
	}
	buffer.WriteString(params.Encode())

	req, err := http.NewRequest(method, URL, buffer)
	if err != nil {
		return nil, nil
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	for key, value := range cookies {
		req.AddCookie(&http.Cookie{Name: key, Value: value})
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}
	return resp, body
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	return r
}

func uploadPhoto(t *testing.T, token, filename string) (int, string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	values := map[string]io.Reader{
		"file":  mustOpen(filename), 
		"token": strings.NewReader(token),
	}

	for key, r := range values {
		var fw io.Writer
		var err error
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}

		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				t.Fatalf("Error CreateFormFile: %v", err)
			}
		} else {
			if fw, err = w.CreateFormField(key); err != nil {
				t.Fatalf("Error CreateFormField: %v", err)	
			}
		}
		if _, err := io.Copy(fw, r); err != nil {
			t.Fatalf("Error io.Copy: %v", err)
		}
	}
	w.Close()

	req, err := http.NewRequest("POST", baseURL + "upload", &b)
	if err != nil {
		return 0, ""
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, ""
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, ""
	}
	return resp.StatusCode, string(body)
}


func testUploadSuccess(token string) func(t *testing.T) {
	return func(t *testing.T) {
		status_code, body := uploadPhoto(t, token, "./sample.jpg")
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

		imageIDsByToken[token] = append(imageIDsByToken[token], value["result"].(string))
	}
}

func testUploadFail(token string) func(t *testing.T) {
	return func(t *testing.T) {
		status_code, body := uploadPhoto(t, token, "./sample.jpg")
		if status_code != 200 {
			t.Fatalf("Status code doesn't equal 200")
		}

		var value map[string]interface{}
		if err := json.Unmarshal([]byte(body), &value); err != nil {
			t.Fatalf("Error parsing json: %v", err)
		}

		if value["error"].(string) != "invalid_token" {
			t.Fatalf("Failed uploading photo: %v", value["error"].(string))
		}
	}
}

func checkImagesSuccess(t *testing.T, list []string) {
	if list == nil {
		t.Fatalf("Empty list")
	}

	for _, image_id := range list {
		resp, body := request(baseURL + "image/" + image_id, "GET", nil, nil)
		if resp == nil || resp.StatusCode != 200 || len(body) < 1000 {
			t.Fatalf("Error getting image")
		}
	}
}

func checkImagesFail(t *testing.T, list []string) {
	if list == nil {
		t.Fatalf("Empty list")
	}

	for _, image_id := range list {
		resp, _ := request(baseURL + "image/" + image_id, "GET", nil, nil)
		if resp == nil || resp.StatusCode != 404 {
			t.Fatalf("Error getting image")
		}
	}
}

func getUnixTime(shift time.Duration) string {
	return strconv.FormatInt(time.Now().Add(shift).Unix(), 10)
}

func TestAll(t *testing.T) {
	go main()
	time.Sleep(1000 * time.Millisecond)

	t.Run("Test uploading with invalid token", testUploadFail(""))
	t.Run("Test uploading with invalid token", testUploadFail("12345"))

	// t.Run("Test creating token", testCreateToken)
	// t.Run("Test editing token", testEditToken)
	// t.Run("Test deleting token", testDeleteToken)

	// t.Run("Test creating token", testCreateToken)
	createToken(t, login(t), "test_token", "1234", "", getUnixTime(1000 * time.Hour))
	t.Run("Benchmark uploading photos", benchUploading)
	t.Run("Benchmark updating data", benchUpdating)

	ctx, cancel := context.WithTimeout(context.Background(), 100 * time.Millisecond)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("Failed stopping server: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
}