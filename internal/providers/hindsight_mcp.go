package providers

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/memoryhindsight"
)

const (
	// hindsightProjectServer is the native MCP name for the project bank.
	hindsightProjectServer = "hindsight-project"
	// hindsightUserServer is the native MCP name for the user bank.
	hindsightUserServer = "hindsight-user"
)

// MCPServer is one Hindsight memory bank exposed as an HTTP MCP endpoint.
// The transport is streamable HTTP. Hindsight mounts that transport at
// /mcp/{bank_id}/ on the API server, with a trailing slash.
type MCPServer struct {
	// Name is the server key written into the agent config.
	Name string
	// Bank is the Hindsight bank id from the memory-hindsight naming rules.
	Bank string
	// URL is the per-bank endpoint, including the trailing slash.
	URL string
}

// HindsightServers builds the project and user MCP servers.
// Disabled settings return no servers and a nil error, so compilers do not
// invent an endpoint. Both banks are returned: kb index retains into the
// project bank, and recall reads the project bank and the user bank. Each
// Hindsight MCP connection is scoped to the bank in the URL.
func HindsightServers(settings hindsight.Settings, id memoryhindsight.Identity) ([]MCPServer, error) {
	if !settings.Enabled {
		return nil, nil
	}
	resolved, err := settings.Resolve()
	if err != nil {
		return nil, fmt.Errorf("hindsight mcp: %w", err)
	}
	base := strings.TrimSpace(settings.APIURL)
	if base == "" {
		base = resolved.APIURL()
	}
	names, err := memoryhindsight.ResolveBanks(id, settings.Banks)
	if err != nil {
		return nil, fmt.Errorf("hindsight mcp: %w", err)
	}
	projectURL, err := hindsightMCPURL(base, names.Project)
	if err != nil {
		return nil, fmt.Errorf("hindsight mcp: %w", err)
	}
	userURL, err := hindsightMCPURL(base, names.User)
	if err != nil {
		return nil, fmt.Errorf("hindsight mcp: %w", err)
	}
	return []MCPServer{
		{Name: hindsightProjectServer, Bank: names.Project, URL: projectURL},
		{Name: hindsightUserServer, Bank: names.User, URL: userURL},
	}, nil
}

func hindsightMCPURL(base, bank string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid hindsight api url")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", fmt.Errorf("hindsight api url must not include a path, query, or fragment")
	}
	if !memoryhindsight.ValidBankID(bank) {
		return "", fmt.Errorf("invalid hindsight bank %q", bank)
	}
	parsed.Path = "/mcp/" + bank + "/"
	parsed.RawPath = ""
	return parsed.String(), nil
}

func orderedMCPServers(servers []MCPServer) ([]MCPServer, error) {
	if len(servers) == 0 {
		return nil, nil
	}
	ordered := append([]MCPServer(nil), servers...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})
	seen := make(map[string]bool, len(ordered))
	for _, server := range ordered {
		if server.Name != hindsightProjectServer && server.Name != hindsightUserServer {
			return nil, fmt.Errorf("hindsight mcp: unknown server %q", server.Name)
		}
		if seen[server.Name] {
			return nil, fmt.Errorf("hindsight mcp: duplicate server %q", server.Name)
		}
		seen[server.Name] = true
		if strings.TrimSpace(server.URL) == "" || strings.ContainsAny(server.URL, "\r\n\x00") {
			return nil, fmt.Errorf("hindsight mcp: server %q has an invalid url", server.Name)
		}
	}
	return ordered, nil
}

func mcpOutputPath(providerName string, provider *Provider, defaultPath string) (string, bool, error) {
	if provider == nil {
		return "", false, fmt.Errorf("%s provider configuration cannot be nil", providerName)
	}
	if provider.Features == nil {
		return defaultPath, true, nil
	}
	feature, ok := provider.Features["mcp"]
	if !ok || feature == nil {
		return defaultPath, true, nil
	}
	if !feature.Enabled {
		return "", false, nil
	}
	path := strings.TrimSpace(feature.Path)
	if path == "" {
		return defaultPath, true, nil
	}
	return path, true, nil
}
