package main

import (
	"fmt"
	"net/http"
	"time"
)

var hc = http.Client{
	Timeout: 500 * time.Millisecond,
}

func doGet(url string) error {
	resp, err := hc.Get(url)
	if err != nil {
		return fmt.Errorf("pingSvc: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return fmt.Errorf("pingSvc: response failed with status code %d", resp.StatusCode)
	}
	return nil
}

func pingSvc(url string, queue <-chan struct{}, errc chan<- error) {
	for range queue {
		errc <- doGet(url)
	}
}
