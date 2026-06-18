package testutil

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var invalidScopeChar = regexp.MustCompile(`[^a-z0-9-]+`)

// RunScope carries a unique test-run suffix for collision-free integration data.
type RunScope struct {
	ID string
}

// NewRunScope builds a sanitized scope from a business scenario slug and CI run id.
func NewRunScope(t TB, slug string) RunScope {
	t.Helper()
	base := sanitizeScope(slug)
	ciRun := sanitizeScope(os.Getenv("GITHUB_RUN_ID"))
	if ciRun == "" {
		ciRun = "local"
	}
	id := fmt.Sprintf("%s-%s-%d", base, ciRun, time.Now().UnixNano())
	t.Logf("integration run scope: %s", id)
	return RunScope{ID: id}
}

// App returns a valid scoped application id.
func (s RunScope) App(base string) string {
	return scopedIdentifier(base, shortScope(s.ID), 30)
}

// Env returns a valid scoped environment id.
func (s RunScope) Env(base string) string {
	return scopedIdentifier(base, shortScope(s.ID), 30)
}

// Key returns the caller-provided element key because scenarios choose valid business keys explicitly.
func (s RunScope) Key(base string) string {
	return base
}

// ClientID returns a valid scoped client instance id.
func (s RunScope) ClientID(base string) string {
	return scopedIdentifier(base, shortScope(s.ID), 64)
}

// Account returns a scoped local test account email.
func (s RunScope) Account(local string) string {
	return fmt.Sprintf("%s.%s@cassem.local", sanitizeScope(local), shortScope(s.ID))
}

// TTLKey returns a namespaced storage key for low-level TTL tests.
func (s RunScope) TTLKey(parts ...string) string {
	cleaned := []string{"cassem", "integration"}
	for _, part := range parts {
		cleaned = append(cleaned, sanitizeScope(part))
	}
	cleaned = append(cleaned, shortScope(s.ID))
	return strings.Join(cleaned, "/")
}

func sanitizeScope(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = invalidScopeChar.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "scope"
	}
	return value
}

func scopedIdentifier(base string, suffix string, max int) string {
	base = sanitizeScope(base)
	reserved := len(suffix) + 1
	if reserved >= max {
		return trimIdentifier(suffix, max)
	}
	baseMax := max - reserved
	return trimIdentifier(base, baseMax) + "-" + suffix
}

func trimIdentifier(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return strings.Trim(value[:max], "-")
}

func shortScope(id string) string {
	if len(id) <= 12 {
		return id
	}
	return strings.Trim(id[len(id)-12:], "-")
}
