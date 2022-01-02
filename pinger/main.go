package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/ping", ping)
	http.HandleFunc("/svc", svcCheck)
	http.HandleFunc("/direct", directCheck)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
