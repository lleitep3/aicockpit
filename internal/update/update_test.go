package update

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckForUpdates_NewVersionAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"tag_name": "v0.2.0",
			"name": "Release v0.2.0",
			"body": "## New Features\n- Feature 1\n- Feature 2",
			"html_url": "https://github.com/lleitep3/aicockpit/releases/tag/v0.2.0"
		}`))
	}))
	defer server.Close()

	service := NewUpdateServiceWithClient(server.Client(), server.URL+"/repos/%s/%s/releases/latest")

	latestVersion, url, err := service.CheckForUpdates()
	if err != nil {
		t.Fatalf("CheckForUpdates() error = %v", err)
	}

	if latestVersion != "0.2.0" {
		t.Errorf("CheckForUpdates() latestVersion = %v, want %v", latestVersion, "0.2.0")
	}

	if url == "" {
		t.Error("CheckForUpdates() url should not be empty")
	}
}

func TestCheckForUpdates_NoNewVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"tag_name": "v0.1.0",
			"name": "Release v0.1.0",
			"body": "## New Features\n- Feature 1",
			"html_url": "https://github.com/lleitep3/aicockpit/releases/tag/v0.1.0"
		}`))
	}))
	defer server.Close()

	service := NewUpdateServiceWithClient(server.Client(), server.URL+"/repos/%s/%s/releases/latest")

	latestVersion, url, err := service.CheckForUpdates()
	if err != nil {
		t.Fatalf("CheckForUpdates() error = %v", err)
	}

	if latestVersion != "" {
		t.Errorf("CheckForUpdates() latestVersion = %v, want empty", latestVersion)
	}

	if url != "" {
		t.Errorf("CheckForUpdates() url = %v, want empty", url)
	}
}

func TestCheckForUpdates_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	service := NewUpdateServiceWithClient(server.Client(), server.URL+"/repos/%s/%s/releases/latest")

	_, _, err := service.CheckForUpdates()
	if err == nil {
		t.Error("CheckForUpdates() should return error on API failure")
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name     string
		latest   string
		current  string
		expected bool
	}{
		{
			name:     "major version newer",
			latest:   "1.0.0",
			current:  "0.1.0",
			expected: true,
		},
		{
			name:     "minor version newer",
			latest:   "0.2.0",
			current:  "0.1.0",
			expected: true,
		},
		{
			name:     "patch version newer",
			latest:   "0.1.1",
			current:  "0.1.0",
			expected: true,
		},
		{
			name:     "same version",
			latest:   "0.1.0",
			current:  "0.1.0",
			expected: false,
		},
		{
			name:     "older version",
			latest:   "0.1.0",
			current:  "0.2.0",
			expected: false,
		},
		{
			name:     "latest has more parts",
			latest:   "0.1.0.1",
			current:  "0.1.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewUpdateService()
			result := service.isNewerVersion(tt.latest, tt.current)
			if result != tt.expected {
				t.Errorf("isNewerVersion(%v, %v) = %v, want %v", tt.latest, tt.current, result, tt.expected)
			}
		})
	}
}

func TestGetReleaseNotes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"tag_name": "v0.2.0",
			"name": "Release v0.2.0",
			"body": "## New Features\n- Feature 1\n- Feature 2",
			"html_url": "https://github.com/lleitep3/aicockpit/releases/tag/v0.2.0"
		}`))
	}))
	defer server.Close()

	service := NewUpdateServiceWithClient(server.Client(), server.URL+"/repos/%s/%s/releases/latest")

	notes, err := service.GetReleaseNotes("0.2.0")
	if err != nil {
		t.Fatalf("GetReleaseNotes() error = %v", err)
	}

	if !strings.Contains(notes, "Feature 1") {
		t.Errorf("GetReleaseNotes() notes should contain 'Feature 1'")
	}
}

func TestGetReleaseNotes_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	service := NewUpdateServiceWithClient(server.Client(), server.URL+"/repos/%s/%s/releases/latest")

	_, err := service.GetReleaseNotes("0.2.0")
	if err == nil {
		t.Error("GetReleaseNotes() should return error when release not found")
	}
}
