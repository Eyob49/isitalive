package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsValidURL(t *testing.T) {
	ok := isValidURL("https://example.com")
	assert.True(t, ok)

	ok = isValidURL("http://example.com")
	assert.True(t, ok)

	ok = isValidURL("ftp://example.com")
	assert.False(t, ok)

	ok = isValidURL("http:///path")
	assert.False(t, ok)

	ok = isValidURL("")
	assert.False(t, ok)
}

func TestHandlerHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	w := httptest.NewRecorder()
	handlerHealthz(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", string(body))
}

func TestHandlerCheck_InvalidInput(t *testing.T) {
	body := strings.NewReader("{'url':")
	req := httptest.NewRequest(http.MethodPost, "https://example.com", body)
	w := httptest.NewRecorder()
	handlerCheck(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, 400, resp.StatusCode)

	body = strings.NewReader("{\"url\":\"ftp://example.com\"}")
	req = httptest.NewRequest(http.MethodPost, "https://example.com", body)
	w = httptest.NewRecorder()
	handlerCheck(w, req)

	resp = w.Result()
	defer resp.Body.Close()

	assert.Equal(t, 400, resp.StatusCode)

	body = strings.NewReader(`{"url":"https://example.com"} {"url":"https://example.org"}`)
	req = httptest.NewRequest(http.MethodPost, "https://example.com", body)
	w = httptest.NewRecorder()
	handlerCheck(w, req)

	resp = w.Result()
	defer resp.Body.Close()

	assert.Equal(t, 400, resp.StatusCode)

	body = strings.NewReader(`{"url":"https://example.com"} garbage`)
	req = httptest.NewRequest(http.MethodPost, "https://example.com", body)
	w = httptest.NewRecorder()
	handlerCheck(w, req)

	resp = w.Result()
	defer resp.Body.Close()

	assert.Equal(t, 400, resp.StatusCode)
}

func TestHandlerCheck_Success(t *testing.T) {
	type parameter struct {
		URL            string `json:"url"`
		StatusCode     int    `json:"status_code"`
		ResponseTimeMS int64  `json:"response_time_ms"`
	}
	ts := httptest.NewServer(http.HandlerFunc(handlerHealthz))
	defer ts.Close()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"url":"`+ts.URL+`"}`))
	w := httptest.NewRecorder()
	handlerCheck(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	params := parameter{}
	if err := decoder.Decode(&params); err != nil {
		t.Fatalf("error decoding json: %v", err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, ts.URL, params.URL)
	assert.Equal(t, http.StatusOK, params.StatusCode)
	assert.GreaterOrEqual(t, params.ResponseTimeMS, int64(0))
}

func TestHandlerCheck_OversizedBody(t *testing.T) {
	body := `{"url":"https://example.com","padding":"` + strings.Repeat("a", 1<<20) + `"}`
	req := httptest.NewRequest(http.MethodPost, "https://example.com", strings.NewReader(body))
	w := httptest.NewRecorder()
	handlerCheck(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, 413, resp.StatusCode)
}

func handlerServiceUnavailable(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
}

func TestHandlerCheck_ServiceUnavailable(t *testing.T) {
	type parameter struct {
		StatusCode int `json:"status_code"`
	}

	ts := httptest.NewServer(http.HandlerFunc(handlerServiceUnavailable))
	defer ts.Close()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"url":"`+ts.URL+`"}`))
	w := httptest.NewRecorder()
	handlerCheck(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	params := parameter{}
	if err := decoder.Decode(&params); err != nil {
		t.Fatalf("error decoding json: %v", err)
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, http.StatusServiceUnavailable, params.StatusCode)
}

func handlerTimeOut(w http.ResponseWriter, _ *http.Request) {
	time.Sleep(11 * time.Second)
	w.WriteHeader(http.StatusGatewayTimeout)
}

func TestHandlerCheck_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerTimeOut))
	defer ts.Close()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"url":"`+ts.URL+`"}`))
	w := httptest.NewRecorder()
	handlerCheck(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}
