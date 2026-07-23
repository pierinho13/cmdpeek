package configsource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pierinho13/cmdpeek/internal/catalog"
)

const (
	ConfigFileEnvironmentVariable   = "CMDPEEK_CONFIG_FILE"
	GitHubConfigEnvironmentVariable = "CMDPEEK_CONFIG_GITHUB"
	GitHubTokenEnvironmentVariable  = "CMDPEEK_GITHUB_TOKEN"
	FallbackGitHubTokenVariable     = "GITHUB_TOKEN"
	DefaultConfigPath               = ".cmdpeek.yaml"

	githubAPIBaseURL  = "https://api.github.com"
	requestTimeout    = 10 * time.Second
	maximumConfigSize = 10 << 20
)

type githubSource struct {
	Owner string
	Repo  string
	Path  string
	Ref   string
}

type cacheMetadata struct {
	ETag         string    `json:"etag"`
	Source       string    `json:"source"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

// Resolve returns the local path that should be loaded by the catalog package.
// Remote GitHub configurations are refreshed before their cached path is returned.
func Resolve(configFlag string) (string, string, error) {
	if path := strings.TrimSpace(configFlag); path != "" {
		return path, "", nil
	}

	if source := strings.TrimSpace(os.Getenv(GitHubConfigEnvironmentVariable)); source != "" {
		path, warning, err := resolveGitHub(source)
		if err != nil {
			return "", "", err
		}
		return path, warning, nil
	}

	if path := strings.TrimSpace(os.Getenv(ConfigFileEnvironmentVariable)); path != "" {
		return path, "", nil
	}

	return DefaultConfigPath, "", nil
}

func resolveGitHub(rawSource string) (string, string, error) {
	source, err := parseGitHubSource(rawSource)
	if err != nil {
		return "", "", err
	}

	cacheDirectory, err := githubCacheDirectory(rawSource)
	if err != nil {
		return "", "", fmt.Errorf("resolve GitHub config cache: %w", err)
	}

	if err := os.MkdirAll(cacheDirectory, 0o700); err != nil {
		return "", "", fmt.Errorf("create GitHub config cache: %w", err)
	}

	configPath := filepath.Join(cacheDirectory, "config.yaml")
	metadataPath := filepath.Join(cacheDirectory, "metadata.json")
	metadata, _ := readMetadata(metadataPath)

	requestURL := githubContentsURL(source)
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("create GitHub config request: %w", err)
	}

	request.Header.Set("Accept", "application/vnd.github.raw+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "cmdpeek")

	if metadata.ETag != "" {
		request.Header.Set("If-None-Match", metadata.ETag)
	}

	if token := githubToken(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return cachedFallback(configPath, fmt.Errorf("download GitHub config: %w", err))
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotModified {
		if _, err := os.Stat(configPath); err != nil {
			return "", "", fmt.Errorf(
				"GitHub returned 304 but no cached configuration exists: %w",
				err,
			)
		}
		return configPath, "", nil
	}

	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		detail := strings.TrimSpace(string(message))
		if detail == "" {
			detail = response.Status
		}
		return cachedFallback(
			configPath,
			fmt.Errorf("GitHub configuration request failed: %s", detail),
		)
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maximumConfigSize+1))
	if err != nil {
		return cachedFallback(configPath, fmt.Errorf("read GitHub config: %w", err))
	}
	if len(data) > maximumConfigSize {
		return cachedFallback(
			configPath,
			fmt.Errorf("GitHub configuration exceeds %d bytes", maximumConfigSize),
		)
	}

	if err := replaceValidatedConfig(configPath, data); err != nil {
		return cachedFallback(configPath, fmt.Errorf("validate GitHub config: %w", err))
	}

	metadata = cacheMetadata{
		ETag:         response.Header.Get("ETag"),
		Source:       rawSource,
		DownloadedAt: time.Now().UTC(),
	}
	if err := writeMetadata(metadataPath, metadata); err != nil {
		return "", "", fmt.Errorf("write GitHub config metadata: %w", err)
	}

	return configPath, "", nil
}

func parseGitHubSource(value string) (githubSource, error) {
	value = strings.TrimSpace(value)
	separator := strings.Index(value, ":")
	if separator <= 0 || separator == len(value)-1 {
		return githubSource{}, fmt.Errorf(
			"invalid %s value %q; expected owner/repository:path/to/config.yaml@ref",
			GitHubConfigEnvironmentVariable,
			value,
		)
	}

	repository := value[:separator]
	filePart := value[separator+1:]

	repositoryParts := strings.Split(repository, "/")
	if len(repositoryParts) != 2 || repositoryParts[0] == "" || repositoryParts[1] == "" {
		return githubSource{}, fmt.Errorf(
			"invalid GitHub repository %q; expected owner/repository",
			repository,
		)
	}

	ref := ""
	if at := strings.LastIndex(filePart, "@"); at > 0 {
		ref = strings.TrimSpace(filePart[at+1:])
		filePart = filePart[:at]
	}

	filePart = strings.TrimPrefix(strings.TrimSpace(filePart), "/")
	if filePart == "" {
		return githubSource{}, fmt.Errorf("GitHub configuration path cannot be empty")
	}

	return githubSource{
		Owner: repositoryParts[0],
		Repo:  repositoryParts[1],
		Path:  filePart,
		Ref:   ref,
	}, nil
}

func githubContentsURL(source githubSource) string {
	pathParts := strings.Split(source.Path, "/")
	for index := range pathParts {
		pathParts[index] = url.PathEscape(pathParts[index])
	}

	endpoint := fmt.Sprintf(
		"%s/repos/%s/%s/contents/%s",
		githubAPIBaseURL,
		url.PathEscape(source.Owner),
		url.PathEscape(source.Repo),
		strings.Join(pathParts, "/"),
	)

	if source.Ref == "" {
		return endpoint
	}

	return endpoint + "?ref=" + url.QueryEscape(source.Ref)
}

func githubToken() string {
	if token := strings.TrimSpace(os.Getenv(GitHubTokenEnvironmentVariable)); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv(FallbackGitHubTokenVariable))
}

func githubCacheDirectory(source string) (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	digest := sha256.Sum256([]byte(source))
	return filepath.Join(
		root,
		"cmdpeek",
		"github",
		hex.EncodeToString(digest[:]),
	), nil
}

func replaceValidatedConfig(path string, data []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "config-*.yaml")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}

	if _, err := catalog.Load(temporaryPath); err != nil {
		return err
	}

	return os.Rename(temporaryPath, path)
}

func cachedFallback(path string, refreshError error) (string, string, error) {
	if _, err := catalog.Load(path); err == nil {
		return path, fmt.Sprintf(
			"could not refresh the GitHub configuration (%v); using the last valid cached copy",
			refreshError,
		), nil
	}

	return "", "", fmt.Errorf(
		"%w; no valid cached GitHub configuration is available",
		refreshError,
	)
}

func readMetadata(path string) (cacheMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheMetadata{}, err
	}

	var metadata cacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return cacheMetadata{}, err
	}
	return metadata, nil
}

func writeMetadata(path string, metadata cacheMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	temporary, err := os.CreateTemp(filepath.Dir(path), "metadata-*.json")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}

	return os.Rename(temporaryPath, path)
}
