package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestIndex(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	if !strings.Contains(w.Body.String(), "go-httpbin") {
		t.Fatalf("expected go-httpbin in index body")
	}
}

func TestFormsPost(t *testing.T) {
	r, _ := http.NewRequest("GET", "/forms/post", nil)
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	if !strings.Contains(w.Body.String(), `<form method="post" action="/post">`) {
		t.Fatalf("expected <form> in body")
	}
}

func TestGet__Basic(t *testing.T) {
	r, _ := http.NewRequest("GET", "/get", nil)
	r.Host = "localhost"
	r.Header.Set("User-Agent", "test")
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected status code 200, got %d", w.Code)
	}

	var resp *Resp
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal body %s from JSON: %s", w.Body, err)
	}

	if resp.Args.Encode() != "" {
		t.Fatalf("expected empty args, got %s", resp.Args.Encode())
	}
	if resp.Origin != "" {
		t.Fatalf("expected empty origin, got %#v", resp.Origin)
	}
	if resp.URL != "http://localhost/get" {
		t.Fatalf("unexpected url: %#v", resp.URL)
	}

	var headerTests = []struct {
		key      string
		expected string
	}{
		{"Content-Type", ""},
		{"User-Agent", "test"},
	}
	for _, test := range headerTests {
		if resp.Headers.Get(test.key) != test.expected {
			t.Fatalf("expected %s = %#v, got %#v", test.key, test.expected, resp.Headers.Get(test.key))
		}
	}
}

func TestGet__OnlyAllowsGets(t *testing.T) {
	r, _ := http.NewRequest("POST", "/get", nil)
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected HTTP 405, got %d", w.Code)
	}
}

func TestGet__CORSHeadersWithoutRequestOrigin(t *testing.T) {
	r, _ := http.NewRequest("GET", "/get", nil)
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin=*, got %#v", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestGet__CORSHeadersWithRequestOrigin(t *testing.T) {
	r, _ := http.NewRequest("GET", "/get", nil)
	r.Header.Set("Origin", "origin")
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "origin" {
		t.Fatalf("expected Access-Control-Allow-Origin=origin, got %#v", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestGet__CORSHeadersWithOptionsVerb(t *testing.T) {
	r, _ := http.NewRequest("OPTIONS", "/get", nil)
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	var headerTests = []struct {
		key      string
		expected string
	}{
		{"Access-Control-Allow-Origin", "*"},
		{"Access-Control-Allow-Credentials", "true"},
		{"Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS"},
		{"Access-Control-Max-Age", "3600"},
		{"Access-Control-Allow-Headers", ""},
	}
	for _, test := range headerTests {
		if w.Header().Get(test.key) != test.expected {
			t.Fatalf("expected %s = %#v, got %#v", test.key, test.expected, w.Header().Get(test.key))
		}
	}
}

func TestGet__CORSAllowHeaders(t *testing.T) {
	r, _ := http.NewRequest("OPTIONS", "/get", nil)
	r.Header.Set("Access-Control-Request-Headers", "X-Test-Header")
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	var headerTests = []struct {
		key      string
		expected string
	}{
		{"Access-Control-Allow-Headers", "X-Test-Header"},
	}
	for _, test := range headerTests {
		if w.Header().Get(test.key) != test.expected {
			t.Fatalf("expected %s = %#v, got %#v", test.key, test.expected, w.Header().Get(test.key))
		}
	}
}

func TestGet__XForwardedProto(t *testing.T) {
	var tests = []struct {
		key   string
		value string
	}{
		{"X-Forwarded-Proto", "https"},
		{"X-Forwarded-Protocol", "https"},
		{"X-Forwarded-Ssl", "on"},
	}

	for _, test := range tests {
		r, _ := http.NewRequest("GET", "/get", nil)
		r.Header.Set(test.key, test.value)
		w := httptest.NewRecorder()
		app().ServeHTTP(w, r)

		var resp *Resp
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		if err != nil {
			t.Fatalf("failed to unmarshal body %s from JSON: %s", w.Body, err)
		}

		if !strings.HasPrefix(resp.URL, "https://") {
			t.Fatalf("%s=%s should result in https URL", test.key, test.value)
		}
	}
}

func TestIP(t *testing.T) {
	r, _ := http.NewRequest("GET", "/ip", nil)
	r.RemoteAddr = "192.168.0.100"
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	var resp *IPResp
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal body %s from JSON: %s", w.Body, err)
	}

	if resp.Origin != r.RemoteAddr {
		t.Fatalf("%#v != %#v", resp.Origin, r.RemoteAddr)
	}
}

func TestUserAgent(t *testing.T) {
	r, _ := http.NewRequest("GET", "/user-agent", nil)
	r.Header.Set("User-Agent", "test")
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	var resp *UserAgentResp
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal body %s from JSON: %s", w.Body, err)
	}

	if resp.UserAgent != "test" {
		t.Fatalf("%#v != \"test\"", resp.UserAgent)
	}
}

func TestHeaders(t *testing.T) {
	r, _ := http.NewRequest("GET", "/headers", nil)
	r.Header.Set("User-Agent", "test")
	r.Header.Set("Foo-Header", "foo")
	r.Header.Add("Bar-Header", "bar1")
	r.Header.Add("Bar-Header", "bar2")
	w := httptest.NewRecorder()
	app().ServeHTTP(w, r)

	var resp *HeadersResp
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal body %s from JSON: %s", w.Body, err)
	}

	for k, expectedValues := range r.Header {
		values, ok := resp.Headers[http.CanonicalHeaderKey(k)]
		if !ok {
			t.Fatalf("expected header %#v in response", k)
		}
		if !reflect.DeepEqual(expectedValues, values) {
			t.Fatalf("%#v != %#v", values, expectedValues)
		}
	}
}