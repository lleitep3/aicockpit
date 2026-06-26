package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/version"
)

const (
	defaultGithubRepoOwner = "lleitep3"
	defaultGithubRepoName  = "aicockpit"
	defaultGithubAPIURL    = "https://api.github.com/repos/%s/%s/releases/latest"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

// UpdateService handles update checking and changelog generation
type UpdateService struct {
	client     *http.Client
	repoOwner  string
	repoName   string
	baseAPIURL string
}

// NewUpdateService creates a new update service
func NewUpdateService() *UpdateService {
	return &UpdateService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		repoOwner:  defaultGithubRepoOwner,
		repoName:   defaultGithubRepoName,
		baseAPIURL: defaultGithubAPIURL,
	}
}

// NewUpdateServiceWithClient creates a new update service with custom HTTP client (for testing)
func NewUpdateServiceWithClient(client *http.Client, baseURL string) *UpdateService {
	return &UpdateService{
		client:     client,
		repoOwner:  defaultGithubRepoOwner,
		repoName:   defaultGithubRepoName,
		baseAPIURL: baseURL,
	}
}

// CheckForUpdates checks if a new version is available on GitHub
func (s *UpdateService) CheckForUpdates() (string, string, error) {
	currentVersion := version.GetVersion()

	url := fmt.Sprintf(s.baseAPIURL, s.repoOwner, s.repoName)

	resp, err := s.client.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", "", fmt.Errorf("failed to parse release info: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")

	if s.isNewerVersion(latestVersion, currentVersion) {
		return latestVersion, release.HTMLURL, nil
	}

	return "", "", nil
}

// isNewerVersion compares two version strings and returns true if the second is newer
func (s *UpdateService) isNewerVersion(latest, current string) bool {
	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		var latestNum, currentNum int
		fmt.Sscanf(latestParts[i], "%d", &latestNum)
		fmt.Sscanf(currentParts[i], "%d", &currentNum)

		if latestNum > currentNum {
			return true
		} else if latestNum < currentNum {
			return false
		}
	}

	return len(latestParts) > len(currentParts)
}

// GetReleaseNotes fetches the release notes for a specific version
func (s *UpdateService) GetReleaseNotes(version string) (string, error) {
	// Build URL for specific tag
	baseURL := strings.Replace(s.baseAPIURL, "/latest", "/tags/v"+version, 1)
	url := fmt.Sprintf(baseURL, s.repoOwner, s.repoName)

	resp, err := s.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch release notes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("failed to parse release info: %w", err)
	}

	return release.Body, nil
}
