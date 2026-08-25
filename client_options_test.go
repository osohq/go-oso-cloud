package oso

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// recordingRoundTripper is a minimal http.RoundTripper that never touches the
// network: it records how many requests it saw and returns a canned
// response. Used to prove that a custom http.Client (and, transitively, its
// Transport) is actually the one used to make requests.
type recordingRoundTripper struct {
	calls int32
}

func (rt *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	atomic.AddInt32(&rt.calls, 1)
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"message":"ok"}`)),
		Request:    req,
	}, nil
}

func TestNewClientWithOptionsUsesCustomHttpClient(t *testing.T) {
	rt := &recordingRoundTripper{}
	customClient := &http.Client{Transport: rt}

	o := NewClientWithOptions("http://example.invalid", "e_0123456789_12345_osotesttoken01xiIn", ClientOptions{
		HttpClient: customClient,
	})

	if err := o.Insert(NewFact("has_role", NewValue("User", "alice"), String("owner"), NewValue("Repo", "acme"))); err != nil {
		t.Fatalf("expected Insert to succeed against the custom client, got error: %v", err)
	}

	if calls := atomic.LoadInt32(&rt.calls); calls != 1 {
		t.Errorf("expected custom http.Client's Transport to be used exactly once, was used %d times", calls)
	}
}

func TestNewClientWithOptionsUsesCustomHttpClientForFallback(t *testing.T) {
	rt := &recordingRoundTripper{}
	customClient := &http.Client{Transport: rt}

	o := NewClientWithOptions("http://example.invalid", "e_0123456789_12345_osotesttoken01xiIn", ClientOptions{
		FallbackUrl: "http://fallback.invalid",
		HttpClient:  customClient,
	}).(OsoClientImpl)

	if o.fallbackHttpClient != customClient {
		t.Error("expected fallback client to reuse the custom http.Client when one is provided")
	}
}

func TestNewClientWithOptionsDefaultsHttpClient(t *testing.T) {
	o := NewClientWithOptions("http://example.invalid", "e_0123456789_12345_osotesttoken01xiIn", ClientOptions{}).(OsoClientImpl)

	if o.httpClient == nil {
		t.Error("expected a default http.Client to be configured when none is provided")
	}
}
