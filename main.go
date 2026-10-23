package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, map[string]string{"error": msg})
}

func isValidURL(input string) bool {
	u, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(u.Scheme)

	return scheme == "http" || scheme == "https" && u.Host != ""
}

func handlerHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handlerCheck(w http.ResponseWriter, r *http.Request) {
	type parameter struct {
		URL string `json:"url"`
	}

	if r.Method != http.MethodPost {
		respondWithError(w, 405, "method not allowed")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameter{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, 400, "error decoding json")
		return
	}

	if !isValidURL(params.URL) {
		respondWithError(w, 400, "invalid URL")
		return
	}

	client := &http.Client{}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		respondWithError(w, 500, "failed to create a request")
		return
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		respondWithError(w, 400, "request failed or timed out")
		return
	}
	defer resp.Body.Close()
	endTime := time.Since(startTime)

	respondWithJSON(w, 200, map[string]any{
		"url":              params.URL,
		"status_code":      resp.StatusCode,
		"response_time_ms": endTime.Milliseconds(),
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlerHealthz)
	mux.HandleFunc("/check", handlerCheck)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server running on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
