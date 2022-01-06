package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func newLocalListener() (net.Listener, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			return nil, fmt.Errorf("httptest: failed to listen on a port: %v", err)
		}
	}
	return l, nil
}

func extractPort(addr net.Addr) (int, error) {
	a := strings.Split(addr.String(), ":")
	if len(a) < 1 {
		return -1, fmt.Errorf("cannot find a listen port")
	}
	pVal := a[len(a)-1]
	port, err := strconv.Atoi(pVal)
	if err != nil {
		return -1, err
	}
	if port <= 0 {
		return -1, fmt.Errorf("port is not a positive int")
	}
	return port, nil
}

func TestDirectCheckHandler(t *testing.T) {
	l, err := newLocalListener()
	if err != nil {
		t.Fatal(err.Error())
	}
	port, err := extractPort(l.Addr())
	if err != nil {
		t.Fatal(err.Error())
	}
	respTS := &httptest.Server{
		Listener: l,
		Config:   &http.Server{Handler: http.HandlerFunc(ping)},
	}
	respTS.Start()
	defer respTS.Close()

	ts := httptest.NewServer(http.HandlerFunc(directCheckWrapper(port)))
	defer ts.Close()

	payload := []directReqPayloadItem{
		{"localhost", []string{"127.0.0.1", "::1"}},
	}
	b, err := json.Marshal(&payload)
	if err != nil {
		t.Error(err.Error())
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, bytes.NewBuffer(b))
	if err != nil {
		t.Error(err.Error())
	}
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Error(err.Error())
	}

	// debug
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err.Error())
	}
	res.Body.Close()
	fmt.Printf("%s", body)
	if err != nil {
		t.Error(err.Error())
	}

}
