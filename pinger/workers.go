package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/denis-zakharov/k8s-net-test/model"
)

const reqBound = 1000

var hc = http.Client{
	Timeout: 1 * time.Second,
	Transport: &http.Transport{
		DisableKeepAlives: true,
	},
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

func pingDirect(queue <-chan model.DirectRespPayloadItem, resc chan<- model.DirectRespPayloadItem, port int) {
	for v := range queue {
		a := v.Addr
		var url string
		if strings.Contains(a, ":") {
			// ipv6
			url = fmt.Sprintf("http://[%s]:%d/ping", a, port)
		} else {
			url = fmt.Sprintf("http://%s:%d/ping", a, port)
		}

		err := doGet(url)
		if err != nil {
			v.Error = err.Error()
		}

		resc <- v
	}
}
