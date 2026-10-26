package memoryhindsight

import (
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/memory"
)

const (
	projectPrefix = "project-"
	userPrefix    = "user-"
	maxBankLen    = 64
)

var bankPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// Names are the Hindsight bank ids used for one project and one user.
type Names struct {
	// Project receives retained knowledge-base documents.
	Project string
	// User receives memories stored with the user bank.
	User string
}

// Identity is the local project and user used to derive bank ids.
type Identity struct {
	// WorkDir is the project directory. Empty falls back to a generic name.
	WorkDir string
	// Username is the operating-system user. Empty falls back to "unknown".
	Username string
}

func (n Names) validate() error {
	if err := checkBank(n.Project); err != nil {
		return fmt.Errorf("project bank: %w", err)
	}
	if err := checkBank(n.User); err != nil {
		return fmt.Errorf("user bank: %w", err)
	}
	if n.Project == n.User {
		return fmt.Errorf("project and user banks must differ")
	}
	return nil
}

func (n Names) retainBank(bank string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(bank)) {
	case "", "default", "project", n.Project:
		return n.Project, nil
	case "user", n.User:
		return n.User, nil
	default:
		return "", memory.ErrInvalidBank
	}
}

func (n Names) queryBanks(bank string) ([]string, error) {
	switch strings.ToLower(strings.TrimSpace(bank)) {
	case "", "default":
		return []string{n.Project, n.User}, nil
	case "project", n.Project:
		return []string{n.Project}, nil
	case "user", n.User:
		return []string{n.User}, nil
	default:
		return nil, memory.ErrInvalidBank
	}
}

// ResolveBanks builds the project and user bank ids.
// Configured names are sanitized. Empty names are derived from id.
func ResolveBanks(id Identity, cfg hindsight.BanksConfig) (Names, error) {
	project := configuredOrDerived(cfg.Project, projectPrefix, projectPart(id.WorkDir), "workspace")
	userName := configuredOrDerived(cfg.User, userPrefix, id.Username, "unknown")
	names := Names{Project: project, User: userName}
	if err := names.validate(); err != nil {
		return Names{}, fmt.Errorf("memory-hindsight banks: %w", err)
	}
	return names, nil
}

// CurrentIdentity reads the working directory and the current user.
func CurrentIdentity() Identity {
	dir, err := os.Getwd()
	if err != nil {
		dir = ""
	}
	name := ""
	if current, err := user.Current(); err == nil {
		name = current.Username
	}
	if strings.TrimSpace(name) == "" {
		name = os.Getenv("USER")
	}
	return Identity{WorkDir: dir, Username: name}
}

func configuredOrDerived(configured, prefix, raw, fallback string) string {
	if strings.TrimSpace(configured) != "" {
		return sanitizeBank(configured)
	}
	part := sanitizePart(raw)
	if part == "" {
		part = fallback
	}
	return fit(prefix, part)
}

func projectPart(dir string) string {
	if mod := moduleName(dir); mod != "" {
		return mod
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	base := filepath.Base(dir)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

func moduleName(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "module ") {
			continue
		}
		mod := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "module ")), `"`)
		return path.Base(mod)
	}
	return ""
}

func sanitizeBank(raw string) string {
	return fit("", sanitizePart(raw))
}

func sanitizePart(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	prevDash := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			prevDash = false
		case b.Len() > 0 && !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-_")
}

func fit(prefix, part string) string {
	id := prefix + part
	if len(id) <= maxBankLen {
		return id
	}
	keep := maxBankLen - len(prefix)
	if keep < 1 {
		return prefix[:maxBankLen]
	}
	part = part[:keep]
	return prefix + strings.Trim(part, "-_")
}

func checkBank(bank string) error {
	if !bankPattern.MatchString(bank) {
		return fmt.Errorf("%w", memory.ErrInvalidBank)
	}
	switch bank {
	case "default", "project", "user":
		return fmt.Errorf("%w: %s is reserved", memory.ErrInvalidBank, bank)
	default:
		return nil
	}
}

func validateAPIURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return fmt.Errorf("invalid hindsight api url")
	}
	return nil
}
