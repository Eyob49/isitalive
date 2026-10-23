package main

import (
	"log"
	"net/http"
)

func handlerHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlerHealthz)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server running on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
