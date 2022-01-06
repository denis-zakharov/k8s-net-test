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

	queue := make(chan struct{}, reqBound)
	errc := make(chan error)

	go func() {
		v := struct{}{}
		for i := 0; i < count; i++ {
			queue <- v
		}
	}()

	for i := 0; i < reqBound; i++ {
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
				log.Printf("[SVC ERROR] %s", err.Error())
			}
			wg.Done()
		}()
	}

	wg.Wait()
	close(queue)
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

func directCheckWrapper(port int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		directCheck(w, r, port)
	}
}

// directCheck verifies pod-to-pod requests
func directCheck(w http.ResponseWriter, r *http.Request, port int) {
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
	r.Body.Close()

	queue := make(chan directRespPayloadItem, reqBound)

	go func() {
		for _, reqItem := range payload {
			for _, a := range reqItem.Addrs {
				v := directRespPayloadItem{
					SrcHost: hostName,
					DstHost: reqItem.Hostname,
					Addr:    a,
				}
				queue <- v
			}
		}
		close(queue)
	}()

	var wg sync.WaitGroup
	wg.Add(reqBound)
	resc := make(chan directRespPayloadItem)
	for i := 0; i < reqBound; i++ {
		go func() {
			pingDirect(queue, resc, port)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(resc)
	}()

	directRespPayload := make([]directRespPayloadItem, 0)
	for res := range resc {
		if res.Error != "" {
			directRespPayload = append(directRespPayload, res)
		}
	}

	b, err := json.Marshal(directRespPayload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot serialize response: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	jsonify(w, b, http.StatusAccepted)
}
