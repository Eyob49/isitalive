package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CheckResponse struct {
	URL            string `json:"url"`
	StatusCode     int    `json:"status_code"`
	ResponseTimeMS int64  `json:"response_time_ms"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

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
	respondWithJSON(w, code, ErrorResponse{Error: msg})
}

func isValidURL(input string) bool {
	u, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(u.Scheme)

	return (scheme == "http" || scheme == "https") && u.Hostname() != ""
}

func handlerHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

var client = &http.Client{
	Timeout: 10 * time.Second,
}

func handlerCheck(w http.ResponseWriter, r *http.Request) {
	type parameter struct {
		URL string `json:"url"`
	}

	const maxBytes = 1 << 20

	if r.Method != http.MethodPost {
		respondWithError(w, 405, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	params := parameter{}

	if err := decoder.Decode(&params); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			respondWithError(w, 413, "Payload too large (max 1 MB)")
			return
		}
		respondWithError(w, 400, "error decoding json")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			respondWithError(w, 400, "body must contain only a single JSON value")
			return
		}

		respondWithError(w, 400, "invalid data after JSON value")
		return
	}

	if !isValidURL(params.URL) {
		respondWithError(w, 400, "invalid URL")
		return
	}

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
		if errors.Is(err, context.DeadlineExceeded) {
			respondWithError(w, 504, "timed out")
			return
		}
		respondWithError(w, 502, "request failed")
		return
	}
	defer resp.Body.Close()
	endTime := time.Since(startTime)

	respondWithJSON(w, http.StatusOK, CheckResponse{
		URL:            params.URL,
		StatusCode:     resp.StatusCode,
		ResponseTimeMS: endTime.Milliseconds(),
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
