package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/staterijndael/http-multiplexer/handler"
)

func TestProcessHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handler.ProcessHandler))
	defer server.Close()

	t.Run("NormalCase", func(t *testing.T) {
		// Sending 100 requests simultaneously
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				reqBody, _ := json.Marshal(map[string]interface{}{
					"urls": []string{"https://rickandmortyapi.com/api/character"},
				})
				req, err := http.NewRequest("POST", server.URL+"/process", bytes.NewBuffer(reqBody))
				if err != nil {
					t.Errorf("Failed to create request: %v", err)
					return
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Errorf("Request failed: %v", err)
					return
				}
				defer resp.Body.Close()

				// Check if the response status is StatusOK
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected StatusOK, got %d", resp.StatusCode)
					return
				}

				// Decode the response body
				var responseBody map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&responseBody)
				if err != nil {
					t.Errorf("Failed to decode response body: %v", err)
					return
				}

				// Check if the response body contains data
				if len(responseBody) == 0 {
					t.Errorf("Expected non-empty response body")
					return
				}
			}()
		}
		wg.Wait()
	})

	errStatusCodeChan := make(chan int, 150)

	t.Run("TooManyRequests", func(t *testing.T) {
		// Sending 100 requests simultaneously
		var wg sync.WaitGroup
		for i := 0; i < 150; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				reqBody, _ := json.Marshal(map[string]interface{}{
					"urls": []string{"https://rickandmortyapi.com/api/character"},
				})
				req, err := http.NewRequest("POST", server.URL+"/process", bytes.NewBuffer(reqBody))
				if err != nil {
					t.Errorf("Failed to create request: %v", err)
					return
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Errorf("Request failed: %v", err)
					return
				}
				defer resp.Body.Close()

				// Check if the response status is StatusOK
				if resp.StatusCode != http.StatusOK {
					if resp.StatusCode != http.StatusTooManyRequests {
						t.Errorf("Expected StatusOK or TooManyRequests, got %d", resp.StatusCode)
						return
					}
					errStatusCodeChan <- resp.StatusCode
					return
				}
			}()
		}
		wg.Wait()

		if len(errStatusCodeChan) == 0 {
			t.Errorf("Expected at least one TooManyRequests error")
			return
		}
	})
}
