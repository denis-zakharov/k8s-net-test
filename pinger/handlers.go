package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

var hostInfo []byte

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
	hostName, err := os.Hostname()
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
	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")
	w.Write(payload)
}

func ping(w http.ResponseWriter, r *http.Request) {
	jsonify(w, hostInfo, http.StatusOK)
}

func mping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST method is required", http.StatusBadRequest)
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read request body", http.StatusInternalServerError)
		return
	}

	for k, v := range r.Header {
		fmt.Fprintf(w, "%s: %s\n", k, v)
	}
	fmt.Fprint(w, string(b))
}
