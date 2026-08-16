package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithCORS(t *testing.T) {
	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := withCORS(next, "https://example.com")

	t.Run("sets headers and passes through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

		assert.True(t, called)
		assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", rec.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("OPTIONS short-circuits", func(t *testing.T) {
		called = false
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("OPTIONS", "/", nil))

		assert.False(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestWithRequestLog(t *testing.T) {
	oldWriter := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(oldWriter)

	t.Run("records explicit status", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		rec := httptest.NewRecorder()
		withRequestLog(next).ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("defaults to 200", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

		rec := httptest.NewRecorder()
		withRequestLog(next).ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestStatusRecorder(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	rec.WriteHeader(http.StatusCreated)

	assert.Equal(t, http.StatusCreated, rec.status)
	assert.Equal(t, http.StatusCreated, rec.ResponseWriter.(*httptest.ResponseRecorder).Code)
}
