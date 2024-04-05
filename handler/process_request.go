package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Request struct {
	Urls []string `json:"urls"`
}

const (
	maxURLs             = 20
	maxConcurrentReqs   = 100
	maxConcurrentOutReq = 4
	timeout             = 1 * time.Second
)

var (
	reqSemaphore = make(chan struct{}, maxConcurrentReqs)
)

func ProcessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	select {
	case reqSemaphore <- struct{}{}:
		defer func() {
			<-reqSemaphore
		}()
	default:
		http.Error(w, "Max http requests on server reached, try again...", http.StatusTooManyRequests)
		return
	}

	request := &Request{}

	decoder := json.NewDecoder(r.Body)
	decoder.Decode(request)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var wg sync.WaitGroup
	result := make(chan map[string]interface{}, len(request.Urls))
	outSemaphore := make(chan struct{}, maxConcurrentOutReq)

	for _, url := range request.Urls {
		outSemaphore <- struct{}{}
		wg.Add(1)
		go func(url string) {
			defer func() {
				<-outSemaphore
				wg.Done()
			}()

			select {
			case <-ctx.Done():
				return
			default:
				data, err := fetchData(url, ctx)
				if err != nil {
					cancel()
					http.Error(w, fmt.Sprintf("Error retreiving data from %s, error: %s", url, err.Error()), http.StatusBadRequest)
					return
				}
				result <- map[string]interface{}{url: data}
			}
		}(url)
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	var responseData = make(map[string]interface{})
	for res := range result {
		for k, v := range res {
			responseData[k] = v
		}
	}

	if ctx.Err() != nil {
		return
	}

	jsonResponse, err := json.Marshal(responseData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func fetchData(url string, ctx context.Context) (interface{}, error) {
	client := http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, nil
}
