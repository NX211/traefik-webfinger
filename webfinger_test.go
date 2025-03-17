package webfinger_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	webfinger "github.com/nx211/traefik-webfinger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebFinger(t *testing.T) {
	cfg := webfinger.CreateConfig()
	cfg.Domain = "example.com"

	// Set up sample resources
	cfg.Resources = map[string]webfinger.WebFingerResponse{
		"acct:alice@example.com": {
			Subject: "acct:alice@example.com",
			Aliases: []string{
				"https://example.com/alice",
				"https://example.com/users/alice",
			},
			Links: []webfinger.WebFingerLink{
				{
					Rel:  "http://webfinger.net/rel/profile-page",
					Type: "text/html",
					Href: "https://example.com/alice",
				},
				{
					Rel:  "self",
					Type: "application/activity+json",
					Href: "https://example.com/users/alice",
				},
			},
		},
	}

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusTeapot) // Should never reach here in our tests
	})

	handler, err := webfinger.New(ctx, next, cfg, "webfinger-test")
	require.NoError(t, err)

	// Test 1: Valid WebFinger request
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/.well-known/webfinger?resource=acct:alice@example.com", nil)
	require.NoError(t, err)

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/jrd+json", recorder.Header().Get("Content-Type"))

	var response webfinger.WebFingerResponse
	err = json.NewDecoder(recorder.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "acct:alice@example.com", response.Subject)
	assert.Len(t, response.Links, 2)
	assert.Equal(t, "http://webfinger.net/rel/profile-page", response.Links[0].Rel)
	assert.Equal(t, "self", response.Links[1].Rel)

	// Test 2: Missing resource parameter
	recorder = httptest.NewRecorder()
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "/.well-known/webfinger", nil)
	require.NoError(t, err)

	handler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	// Test 3: Resource not found
	recorder = httptest.NewRecorder()
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "/.well-known/webfinger?resource=acct:bob@example.com", nil)
	require.NoError(t, err)

	handler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	// Test 4: Resource for different domain
	recorder = httptest.NewRecorder()
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "/.well-known/webfinger?resource=acct:alice@otherdomain.com", nil)
	require.NoError(t, err)

	handler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	// Test 5: Non-WebFinger path should be passed through
	recorder = httptest.NewRecorder()
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "/some/other/path", nil)
	require.NoError(t, err)

	handler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusTeapot, recorder.Code)

	// Test 6: Method not allowed
	recorder = httptest.NewRecorder()
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, "/.well-known/webfinger?resource=acct:alice@example.com", nil)
	require.NoError(t, err)

	handler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}

func TestPassthrough(t *testing.T) {
	cfg := webfinger.CreateConfig()
	cfg.Domain = "example.com"
	cfg.Passthrough = true // Enable passthrough for this test

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(rw, `{"message":"backend response"}`)
		require.NoError(t, err)
	})

	handler, err := webfinger.New(ctx, next, cfg, "webfinger-test")
	require.NoError(t, err)

	// Test passthrough when resource not found
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/.well-known/webfinger?resource=acct:unknown@example.com", nil)
	require.NoError(t, err)

	handler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	body, err := io.ReadAll(recorder.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"message":"backend response"}`, string(body))

	// Test passthrough for different domain
	recorder = httptest.NewRecorder()
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "/.well-known/webfinger?resource=acct:user@otherdomain.com", nil)
	require.NoError(t, err)

	handler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *webfinger.Config
		expectError bool
	}{
		{
			name: "Valid config",
			config: &webfinger.Config{
				Domain: "example.com",
				Resources: map[string]webfinger.WebFingerResponse{
					"acct:user@example.com": {
						Subject: "acct:user@example.com",
						Links: []webfinger.WebFingerLink{
							{Rel: "self", Href: "https://example.com/user"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Missing domain",
			config: &webfinger.Config{
				Resources: map[string]webfinger.WebFingerResponse{
					"acct:user@example.com": {
						Subject: "acct:user@example.com",
					},
				},
			},
			expectError: true,
		},
		{
			name: "Resource domain mismatch",
			config: &webfinger.Config{
				Domain: "example.com",
				Resources: map[string]webfinger.WebFingerResponse{
					"acct:user@otherdomain.com": {
						Subject: "acct:user@otherdomain.com",
					},
				},
			},
			expectError: true,
		},
		{
			name: "Missing subject",
			config: &webfinger.Config{
				Domain: "example.com",
				Resources: map[string]webfinger.WebFingerResponse{
					"acct:user@example.com": {
						Links: []webfinger.WebFingerLink{
							{Rel: "self", Href: "https://example.com/user"},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "Missing link rel",
			config: &webfinger.Config{
				Domain: "example.com",
				Resources: map[string]webfinger.WebFingerResponse{
					"acct:user@example.com": {
						Subject: "acct:user@example.com",
						Links: []webfinger.WebFingerLink{
							{Href: "https://example.com/user"},
						},
					},
				},
			},
			expectError: true,
		},
	}

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := webfinger.New(ctx, next, tt.config, "test")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
