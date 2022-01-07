package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/denis-zakharov/k8s-net-test/model"
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

	var reqPayload model.SvcReqPayload
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
		close(queue)
	}()

	var wg sync.WaitGroup
	wg.Add(reqBound)
	for i := 0; i < reqBound; i++ {
		go func() {
			pingSvc(svcUrl, queue, errc)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	errCount := 0
	for err := range errc {
		if err != nil {
			errCount++
			log.Printf("[SVC ERROR] %s", err.Error())
		}
	}

	respPayload := model.SvcRespPayload{
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

	var payload []model.DirectReqPayloadItem
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Decode Failed", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	queue := make(chan model.DirectRespPayloadItem, reqBound)

	go func() {
		for _, reqItem := range payload {
			for _, a := range reqItem.Addrs {
				v := model.DirectRespPayloadItem{
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
	resc := make(chan model.DirectRespPayloadItem)
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

	directRespPayload := make([]model.DirectRespPayloadItem, 0)
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
