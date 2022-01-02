package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/healthz", func(http.ResponseWriter, *http.Request) {})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
