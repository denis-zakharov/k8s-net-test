package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	port := flag.Int("port", 8080, "listen port")
	flag.Parse()

	pVal, ok := os.LookupEnv("PINGER_PORT")
	if ok {
		p, err := strconv.Atoi(pVal)
		if err != nil {
			log.Fatal("PINGER_PORT cannot be converted to integer")
		}
		if p <= 0 {
			log.Fatal("Port is not positive integer")
		}
		port = &p
	}

	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/ping", ping)
	http.HandleFunc("/svc", svcCheck)
	http.HandleFunc("/direct", directCheckWrapper(*port))
	listAddr := fmt.Sprintf(":%d", *port)
	log.Fatal(http.ListenAndServe(listAddr, nil))
}
