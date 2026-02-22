// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Save original env var
	originalToken := os.Getenv(envGitHubToken)
	defer func() {
		if originalToken != "" {
			_ = os.Setenv(envGitHubToken, originalToken)
		} else {
			_ = os.Unsetenv(envGitHubToken)
		}
	}()

	t.Run("without token", func(t *testing.T) {
		_ = os.Unsetenv(envGitHubToken)
		client := NewClient("owner", "repo", false)

		if client.Token != "" {
			t.Errorf("Expected empty token, got %q", client.Token)
		}
		if client.Owner != "owner" {
			t.Errorf("Expected owner 'owner', got %q", client.Owner)
		}
		if client.Repo != "repo" {
			t.Errorf("Expected repo 'repo', got %q", client.Repo)
		}
		if client.Prerelease {
			t.Error("Expected Prerelease to be false")
		}
	})

	t.Run("with token", func(t *testing.T) {
		testToken := "ghp_test_token_123"
		_ = os.Setenv(envGitHubToken, testToken)

		client := NewClient("owner", "repo", true)

		if client.Token != testToken {
			t.Errorf("Expected token %q, got %q", testToken, client.Token)
		}
		if !client.Prerelease {
			t.Error("Expected Prerelease to be true")
		}
	})
}

func TestClient_AuthenticationHeader(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		token         string
		expectAuthHdr bool
	}{
		{
			name:          "with token",
			token:         "ghp_test_token",
			expectAuthHdr: true,
		},
		{
			name:          "without token",
			token:         "",
			expectAuthHdr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that checks for the Authorization header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get(hdrAuthorization)

				if tt.expectAuthHdr {
					expectedAuth := "Bearer " + tt.token
					if authHeader != expectedAuth {
						t.Errorf("Expected Authorization header %q, got %q", expectedAuth, authHeader)
					}
				} else {
					if authHeader != "" {
						t.Errorf("Expected no Authorization header, got %q", authHeader)
					}
				}

				// Return a minimal valid response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"tag_name": "v1.0.0", "published_at": "2023-01-01T00:00:00Z", "draft": false, "prerelease": false, "assets": []}`))
			}))
			defer server.Close()

			client := &Client{
				Owner:      "test",
				Repo:       "test",
				Prerelease: false,
				Token:      tt.token,
			}

			// Override the URL to point to our test server
			// We need to use the get method directly
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			_, err = client.do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
		})
	}
}

func TestClient_RateLimitLogging(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		headers     map[string]string
		expectError bool
	}{
		{
			name: "with rate limit headers",
			headers: map[string]string{
				hdrRateLimit:       "5000",
				hdrRateLimitUsed:   "42",
				hdrRateLimitRemain: "4958",
				hdrRateLimitReset:  "1640000000",
			},
			expectError: false,
		},
		{
			name: "low rate limit remaining",
			headers: map[string]string{
				hdrRateLimit:       "60",
				hdrRateLimitUsed:   "55",
				hdrRateLimitRemain: "5",
				hdrRateLimitReset:  "1640000000",
			},
			expectError: false,
		},
		{
			name:        "without rate limit headers",
			headers:     map[string]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Set rate limit headers
				for key, value := range tt.headers {
					w.Header().Set(key, value)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"tag_name": "v1.0.0", "published_at": "2023-01-01T00:00:00Z", "draft": false, "prerelease": false, "assets": []}`))
			}))
			defer server.Close()

			client := &Client{
				Owner:      "test",
				Repo:       "test",
				Prerelease: false,
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			_, err = client.do(req)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestClient_logRateLimitInfo(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		Owner:      "test",
		Repo:       "test",
		Prerelease: false,
		Token:      "test_token",
	}

	t.Run("with complete headers", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{
				hdrRateLimit:       []string{"5000"},
				hdrRateLimitUsed:   []string{"100"},
				hdrRateLimitRemain: []string{"4900"},
				hdrRateLimitReset:  []string{"1640000000"},
			},
		}

		// Should not panic
		client.logRateLimitInfo(ctx, resp)
	})

	t.Run("with missing headers", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{},
		}

		// Should not panic
		client.logRateLimitInfo(ctx, resp)
	})

	t.Run("with invalid reset time", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{
				hdrRateLimit:       []string{"5000"},
				hdrRateLimitRemain: []string{"4900"},
				hdrRateLimitReset:  []string{"invalid"},
			},
		}

		// Should not panic
		client.logRateLimitInfo(ctx, resp)
	})
}

func TestClient_Headers(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify required headers
		if accept := r.Header.Get(hdrAccept); accept != contentType {
			t.Errorf("Expected Accept header %q, got %q", contentType, accept)
		}
		if apiVersion := r.Header.Get(hdrGitHubAPIVersion); apiVersion != githubAPIVersion {
			t.Errorf("Expected X-GitHub-Api-Version header %q, got %q", githubAPIVersion, apiVersion)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name": "v1.0.0", "published_at": "2023-01-01T00:00:00Z", "draft": false, "prerelease": false, "assets": []}`))
	}))
	defer server.Close()

	client := &Client{
		Owner:      "test",
		Repo:       "test",
		Prerelease: false,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = client.do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestClient_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		statusCode     int
		expectedErrMsg string
	}{
		{
			name:           "404 not found",
			statusCode:     http.StatusNotFound,
			expectedErrMsg: "invalid API status code (404)",
		},
		{
			name:           "403 forbidden",
			statusCode:     http.StatusForbidden,
			expectedErrMsg: "invalid API status code (403)",
		},
		{
			name:           "429 rate limit exceeded",
			statusCode:     http.StatusTooManyRequests,
			expectedErrMsg: "invalid API status code (429)",
		},
		{
			name:           "500 server error",
			statusCode:     http.StatusInternalServerError,
			expectedErrMsg: "invalid API status code (500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := &Client{
				Owner:      "test",
				Repo:       "test",
				Prerelease: false,
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			_, err = client.do(req)
			if err == nil {
				t.Error("Expected error but got none")
			}
			if err != nil && !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedErrMsg, err.Error())
			}
		})
	}
}
