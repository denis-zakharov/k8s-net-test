package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const reqBound = 1000

var hc = http.Client{
	Timeout: 5 * time.Second,
}

func doGet(url string) error {
	resp, err := hc.Get(url)
	if err != nil {
		return fmt.Errorf("doGet: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return fmt.Errorf("doGet: response failed with status code %d", resp.StatusCode)
	}
	return nil
}

func pingSvc(url string, queue <-chan struct{}, errc chan<- error) {
	for range queue {
		errc <- doGet(url)
	}
}

func pingDirect(queue <-chan directRespPayloadItem, resc chan<- directRespPayloadItem) {
	for v := range queue {
		a := v.Addr
		var url string
		if strings.Contains(a, ":") {
			// ipv6
			url = fmt.Sprintf("http://[%s]:8080/ping", a)
		} else {
			url = fmt.Sprintf("http://%s:8080/ping", a)
		}

		err := doGet(url)
		if err != nil {
			v.Error = err.Error()
		}

		resc <- v
	}
}
