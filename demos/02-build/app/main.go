package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "DOUGBR - aplicacao de demonstracao (uid=%d)\n", os.Getuid())
	})
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	log.Println("ouvindo em :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
