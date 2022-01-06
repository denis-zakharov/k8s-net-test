package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
)

const pingBound = 1000

var (
	hostName string
	hostInfo []byte
)

func init() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal("Cannot list network interfaces")
	}

	var localAddrs []string
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalUnicast() {
			//ipnet.IP.To16() != nil for ipv6
			localAddrs = append(localAddrs, ipnet.IP.String())
		}
	}
	instance := make(map[string][]string, 1)
	hostName, err = os.Hostname()
	if err != nil {
		log.Fatal("Cannot resolve hostname")
	}
	instance[hostName] = localAddrs
	hostInfo, err = json.Marshal(instance)
	if err != nil {
		log.Fatal("Cannot serialize hostname-addrs map to JSON")
	}
}

func jsonify(w http.ResponseWriter, payload []byte, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(payload)
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func ping(w http.ResponseWriter, r *http.Request) {
	jsonify(w, hostInfo, http.StatusOK)
}

// svcCheck verifies k8s service
func svcCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST method is required", http.StatusMethodNotAllowed)
		return
	}

	var reqPayload svcReqPayload
	err := json.NewDecoder(r.Body).Decode(&reqPayload)
	if err != nil {
		http.Error(w, "Decode Failed", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	svcUrl := reqPayload.SvcURL
	count := reqPayload.Count

	queue := make(chan struct{}, pingBound)
	errc := make(chan error)

	go func() {
		v := struct{}{}
		for i := 0; i < count; i++ {
			queue <- v
		}
	}()

	for i := 0; i < pingBound; i++ {
		go func() {
			pingSvc(svcUrl, queue, errc)
		}()
	}

	var wg sync.WaitGroup
	wg.Add(count)
	var mu sync.Mutex
	errCount := 0

	for i := 0; i < count; i++ {
		go func() {
			err := <-errc
			if err != nil {
				mu.Lock()
				errCount++
				mu.Unlock()
			}
			wg.Done()
		}()
	}

	wg.Wait()
	close(queue)
	close(errc)
	respPayload := svcRespPayload{
		SrcHost: hostName,
		Errors:  errCount,
	}

	b, err := json.Marshal(respPayload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot serialize response: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	jsonify(w, b, http.StatusAccepted)
}

// directCheck verifies pod-to-pod requests
func directCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST method is required", http.StatusMethodNotAllowed)
		return
	}

	var payload []directReqPayloadItem
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Decode Failed", http.StatusBadRequest)
		return
	}

	fmt.Printf("%#+v\n", payload)

	jsonify(w, []byte{}, http.StatusAccepted)
}
