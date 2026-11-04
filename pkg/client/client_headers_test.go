package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/config"
)

func TestJSONPostHeadersDefaults(t *testing.T) {
	got := jsonPostHeaders("", nil)
	want := map[string]string{
		"Accept":          "application/json",
		"Accept-Language": "en-US,en;q=0.5",
		"Content-Type":    "application/json",
		"Connection":      "keep-alive",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("jsonPostHeaders() = %v, want %v", got, want)
	}
	if _, ok := got["Cookie"]; ok {
		t.Fatal("empty cookie must not produce a Cookie header")
	}
}

func TestJSONPostHeadersIncludesCookie(t *testing.T) {
	got := jsonPostHeaders("A=1; B=2", map[string]string{})
	if got["Cookie"] != "A=1; B=2" {
		t.Fatalf("Cookie = %q", got["Cookie"])
	}
}

func TestJSONPostHeadersMergesExtraHeaders(t *testing.T) {
	extra := map[string]string{"x-yahoo-cg-client-name": "finance"}
	got := jsonPostHeaders("A=1", extra)
	if got["x-yahoo-cg-client-name"] != "finance" {
		t.Fatalf("extra header missing: %v", got)
	}
	if got["Content-Type"] != "application/json" || got["Accept"] != "application/json" || got["Cookie"] != "A=1" {
		t.Fatalf("defaults lost after merge: %v", got)
	}
	if len(got) != 6 {
		t.Fatalf("unexpected header count %d: %v", len(got), got)
	}
}

func TestJSONPostHeadersOverrideIsCaseInsensitive(t *testing.T) {
	got := jsonPostHeaders("A=1", map[string]string{
		"content-type": "application/graphql+json",
		"ACCEPT":       "*/*",
		"cookie":       "B=2",
	})
	for _, name := range []string{"Content-Type", "Accept", "Cookie"} {
		if _, ok := got[name]; ok {
			t.Fatalf("default %q should have been replaced, got %v", name, got)
		}
	}
	if got["content-type"] != "application/graphql+json" || got["ACCEPT"] != "*/*" || got["cookie"] != "B=2" {
		t.Fatalf("overrides not applied: %v", got)
	}
	if len(got) != 5 {
		t.Fatalf("duplicate headers after override: %v", got)
	}
}

func TestMergeHeadersSkipsBlankNamesAndTrims(t *testing.T) {
	got := mergeHeaders(map[string]string{"Accept": "a"}, map[string]string{
		"":          "ignored",
		"   ":       "ignored",
		" X-Trim  ": "v",
	})
	want := map[string]string{"Accept": "a", "X-Trim": "v"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeHeaders() = %v, want %v", got, want)
	}
}

func TestMergeHeadersDoesNotMutateExtra(t *testing.T) {
	extra := map[string]string{"x-test": "1", "Content-Type": "text/plain"}
	snapshot := map[string]string{"x-test": "1", "Content-Type": "text/plain"}
	_ = jsonPostHeaders("A=1", extra)
	if !reflect.DeepEqual(extra, snapshot) {
		t.Fatalf("extra headers mutated: %v", extra)
	}
}

func TestMergeHeadersAllowsEmptyValue(t *testing.T) {
	got := mergeHeaders(map[string]string{"Accept": "a"}, map[string]string{"x-empty": ""})
	if v, ok := got["x-empty"]; !ok || v != "" {
		t.Fatalf("empty-value header should be kept: %v", got)
	}
}

type capturedRequest struct {
	method string
	path   string
	query  string
	header http.Header
	body   string
}

func newCaptureServer(t *testing.T) (*httptest.Server, func() capturedRequest) {
	t.Helper()
	var (
		mu   sync.Mutex
		last capturedRequest
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		last = capturedRequest{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			header: r.Header.Clone(),
			body:   string(body),
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() capturedRequest {
		mu.Lock()
		defer mu.Unlock()
		return last
	}
}

func newLocalTestClient(t *testing.T) *Client {
	t.Helper()
	config.Reset()
	t.Cleanup(config.Reset)
	c, err := New(WithTimeout(10), WithProxy(""))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestPostJSONWithHeadersSendsMergedHeaders(t *testing.T) {
	srv, last := newCaptureServer(t)
	c := newLocalTestClient(t)
	c.SetCookie("A=1")

	resp, err := c.PostJSONWithHeaders(srv.URL+"/graphql", nil, []byte(`{"q":1}`), map[string]string{
		"x-yahoo-cg-client-name": "finance",
	})
	if err != nil {
		t.Fatalf("PostJSONWithHeaders() error: %v", err)
	}
	if resp.StatusCode != http.StatusOK || resp.Body != `{"ok":true}` {
		t.Fatalf("unexpected response: %+v", resp)
	}

	req := last()
	if req.method != http.MethodPost || req.path != "/graphql" {
		t.Fatalf("request = %s %s", req.method, req.path)
	}
	if req.body != `{"q":1}` {
		t.Fatalf("body = %q", req.body)
	}
	if got := req.header.Get("X-Yahoo-Cg-Client-Name"); got != "finance" {
		t.Fatalf("x-yahoo-cg-client-name = %q", got)
	}
	if got := req.header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := req.header.Get("Cookie"); got != "A=1" {
		t.Fatalf("Cookie = %q", got)
	}
	if got := req.header.Values("Content-Type"); len(got) != 1 {
		t.Fatalf("Content-Type sent %d times: %v", len(got), got)
	}
}

func TestPostJSONWithHeadersEncodesParams(t *testing.T) {
	srv, last := newCaptureServer(t)
	c := newLocalTestClient(t)

	if _, err := c.PostJSONWithHeaders(srv.URL+"/", map[string][]string{"crumb": {"abc"}}, []byte(`{}`), nil); err != nil {
		t.Fatalf("PostJSONWithHeaders() error: %v", err)
	}
	if got := last().query; got != "crumb=abc" {
		t.Fatalf("query = %q", got)
	}
}

func TestPostJSONUnchangedWithoutExtraHeaders(t *testing.T) {
	srv, last := newCaptureServer(t)
	c := newLocalTestClient(t)

	if _, err := c.PostJSON(srv.URL+"/", nil, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("PostJSON() error: %v", err)
	}
	req := last()
	if req.header.Get("Content-Type") != "application/json" || req.header.Get("Accept") != "application/json" {
		t.Fatalf("default headers missing: %v", req.header)
	}
	if req.header.Get("X-Yahoo-Cg-Client-Name") != "" {
		t.Fatalf("PostJSON must not send extra headers: %v", req.header)
	}
	if req.header.Get("Cookie") != "" {
		t.Fatalf("no cookie expected: %v", req.header)
	}
}

func TestPostJSONWithHeadersContentTypeOverride(t *testing.T) {
	srv, last := newCaptureServer(t)
	c := newLocalTestClient(t)

	if _, err := c.PostJSONWithHeaders(srv.URL+"/", nil, []byte(`{}`), map[string]string{
		"content-type": "application/graphql-response+json",
	}); err != nil {
		t.Fatalf("PostJSONWithHeaders() error: %v", err)
	}
	got := last().header.Values("Content-Type")
	if len(got) != 1 || got[0] != "application/graphql-response+json" {
		t.Fatalf("Content-Type = %v", got)
	}
}
