package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Eyob49/isitalive/internal/database"
)

type CheckResponse struct {
	ID             string    `json:"id"`
	URL            string    `json:"url"`
	StatusCode     int       `json:"status_code"`
	ResponseTimeMS int64     `json:"response_time_ms"`
	CheckedAt      time.Time `json:"checked_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type apiConfig struct {
	queries *database.Queries
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
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

func (cfg *apiConfig) handlerCheck(w http.ResponseWriter, r *http.Request) {
	type parameter struct {
		URL string `json:"url"`
	}

	const maxBytes = 1 << 20

	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	params := parameter{}

	if err := decoder.Decode(&params); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			respondWithError(w, http.StatusRequestEntityTooLarge, "Payload too large (max 1 MB)")
			return
		}
		respondWithError(w, http.StatusBadRequest, "error decoding json")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			respondWithError(w, http.StatusBadRequest, "body must contain only a single JSON value")
			return
		}

		respondWithError(w, http.StatusBadRequest, "invalid data after JSON value")
		return
	}

	if !isValidURL(params.URL) {
		respondWithError(w, http.StatusBadRequest, "invalid URL")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create a request")
		return
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			respondWithError(w, http.StatusGatewayTimeout, "timed out")
			return
		}
		respondWithError(w, http.StatusBadGateway, "request failed")
		return
	}
	defer resp.Body.Close()
	endTime := time.Since(startTime)

	check, err := cfg.queries.CreateCheck(r.Context(), database.CreateCheckParams{
		Url:            params.URL,
		StatusCode:     int32(resp.StatusCode),
		ResponseTimeMs: endTime.Milliseconds(),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to save check result")
		return
	}

	respondWithJSON(w, http.StatusOK, CheckResponse{
		ID:             check.ID.String(),
		URL:            check.Url,
		StatusCode:     int(check.StatusCode),
		ResponseTimeMS: check.ResponseTimeMs,
		CheckedAt:      check.CheckedAt,
	})
}

func (cfg *apiConfig) handlerGetChecks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, 405, "method not allowed")
		return
	}

	limit := int32(10) // Default Limit
	if l := r.URL.Query().Get("limit"); l != "" {
		parsedLimit, err := strconv.Atoi(l)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			respondWithError(w, http.StatusBadRequest, "invalid limit parameter (must be between 1 and 100) ")
			return
		}
		limit = int32(parsedLimit)
	}

	checks, err := cfg.queries.GetChecks(r.Context(), limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to retrieve checks")
		return
	}

	response := make([]CheckResponse, 0, len(checks))
	for _, check := range checks {
		response = append(response, CheckResponse{
			ID:             check.ID.String(),
			URL:            check.Url,
			StatusCode:     int(check.StatusCode),
			ResponseTimeMS: check.ResponseTimeMs,
			CheckedAt:      check.CheckedAt,
		})
	}
	respondWithJSON(w, http.StatusOK, response)
}

func main() {
	queries, db, err := OpenQueries()
	if err != nil {
		log.Fatalf("Failed to open database queries: %v", err)
	}
	defer db.Close()

	apiCfg := &apiConfig{queries: queries}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlerHealthz)
	mux.HandleFunc("/check", apiCfg.handlerCheck)
	mux.HandleFunc("/checks", apiCfg.handlerGetChecks)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server running on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil {
		log.Printf("HTTP server failed: %v", err)
	}
}
