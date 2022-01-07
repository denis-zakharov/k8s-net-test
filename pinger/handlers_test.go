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

	"github.com/denis-zakharov/k8s-net-test/model"
)

func newLocalListener() (net.Listener, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, fmt.Errorf("httptest: failed to listen on a port: %v", err)

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

func newRespServer() (srv *httptest.Server, port int, err error) {
	l, err := newLocalListener()
	if err != nil {
		return nil, -1, err
	}
	port, err = extractPort(l.Addr())
	if err != nil {
		return nil, -1, err
	}
	srv = &httptest.Server{
		Listener: l,
		Config:   &http.Server{Handler: http.HandlerFunc(ping)},
	}
	srv.Start()
	return srv, port, nil
}

func TestDirectCheckHandler(t *testing.T) {
	respTS, port, err := newRespServer()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer respTS.Close()
	ts := httptest.NewServer(http.HandlerFunc(directCheckWrapper(port)))
	defer ts.Close()

	payload := []model.DirectReqPayloadItem{
		{Hostname: "localhost", Addrs: []string{"127.0.0.1", "::1", "f:a:i:1::1"}},
	}
	b, err := json.Marshal(&payload)
	if err != nil {
		t.Fatal(err.Error())
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, bytes.NewBuffer(b))
	if err != nil {
		t.Fatal(err.Error())
	}
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Error(err.Error())
	}

	// should be one error
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err.Error())
	}
	res.Body.Close()
	var respPayload []model.DirectRespPayloadItem
	err = json.Unmarshal(body, &respPayload)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(respPayload) != 1 {
		t.Errorf("one address should fail: %v", respPayload)
	}
}

func TestSvcHandler(t *testing.T) {
	respTS, _, err := newRespServer()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer respTS.Close()
	ts := httptest.NewServer(http.HandlerFunc(svcCheck))
	defer ts.Close()

	payload := model.SvcReqPayload{SvcURL: respTS.URL, Count: 100}
	b, err := json.Marshal(&payload)
	if err != nil {
		t.Fatal(err.Error())
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, bytes.NewBuffer(b))
	if err != nil {
		t.Fatal(err.Error())
	}
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Error(err.Error())
	}

	// should be zero errors
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err.Error())
	}
	res.Body.Close()
	var respPayload model.SvcRespPayload
	err = json.Unmarshal(body, &respPayload)
	if err != nil {
		t.Fatal(err.Error())
	}

	if respPayload.Errors != 0 {
		t.Errorf("should be no errors: %v", respPayload)
	}
}
