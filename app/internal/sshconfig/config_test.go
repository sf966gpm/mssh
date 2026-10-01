package sshconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	contents := `# comment

Host home work
    HostName "192.168.1.10" # inline comment
    User maxim
    Port 22
    IdentityFile ~/.ssh/home/id_ed25519
    IdentityFile ~/.ssh/home/id_rsa

Host *.example.org !banned.example.org
    User deploy

Include ~/.ssh/work/config.conf
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Hosts) != 2 {
		t.Fatalf("got %d hosts, want 2", len(config.Hosts))
	}
	if got := strings.Join(config.Hosts[0].Patterns, " "); got != "home work" {
		t.Errorf("got patterns %q, want %q", got, "home work")
	}
	if got := config.Hosts[0].Fields[0].DisplayValue(); got != "192.168.1.10" {
		t.Errorf("got displayed HostName %q, want %q", got, "192.168.1.10")
	}
	if got := len(config.Hosts[0].Fields); got != 5 {
		t.Errorf("got %d fields, want 5 including repeated IdentityFile", got)
	}
	if len(config.Includes) != 1 || config.Includes[0].Pattern != "~/.ssh/work/config.conf" {
		t.Errorf("got includes %#v", config.Includes)
	}
}

func TestParseFileReportsUnsupportedContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	contents := "Host before\n    User one\nMatch host *.example.org\n    User two\nUnknownDirective value\nHost after\n    User three\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Hosts) != 2 {
		t.Fatalf("got %d hosts, want 2", len(config.Hosts))
	}
	if got := len(config.Hosts[0].Fields); got != 1 {
		t.Errorf("fields after Match were attributed to the first host: got %d, want 1", got)
	}
	if got := len(config.Hosts[1].Fields); got != 1 {
		t.Errorf("got %d fields for second host, want 1", got)
	}
	if len(config.Diagnostics) != 3 {
		t.Errorf("got %d diagnostics, want 3", len(config.Diagnostics))
	}
}

func TestParseFileMissing(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "missing-config"))
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("got error %v, want missing-config error", err)
	}
}

func TestParseEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Hosts) != 0 || len(config.Diagnostics) != 0 {
		t.Fatalf("got hosts=%d diagnostics=%d, want both zero", len(config.Hosts), len(config.Diagnostics))
	}
}

func TestParseFileExpandedResolvesNestedIncludesInOrder(t *testing.T) {
	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "nested")
	if err := os.Mkdir(nestedDir, 0o700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "config")
	child := filepath.Join(dir, "child.conf")
	nested := filepath.Join(nestedDir, "one.conf")
	writeConfig(t, root, "Host root\nInclude child.conf\nHost after\n")
	writeConfig(t, child, "Host child\nInclude nested/*.conf\n")
	writeConfig(t, nested, "Host nested\n")

	config, err := ParseFileExpanded(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := hostPatterns(config); !strings.EqualFold(got, "root child nested after") {
		t.Errorf("got hosts %q, want %q", got, "root child nested after")
	}
	if len(config.Includes) != 2 {
		t.Fatalf("got %d includes, want 2", len(config.Includes))
	}
	if config.Hosts[1].Path != child || config.Hosts[2].Path != nested {
		t.Errorf("got host sources %q and %q, want %q and %q", config.Hosts[1].Path, config.Hosts[2].Path, child, nested)
	}
	if len(config.Diagnostics) != 0 {
		t.Errorf("got diagnostics %#v, want none", config.Diagnostics)
	}
}

func TestParseFileExpandedReportsMissingInclude(t *testing.T) {
	root := filepath.Join(t.TempDir(), "config")
	writeConfig(t, root, "Host root\nInclude missing.conf\n")

	config, err := ParseFileExpanded(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Diagnostics) != 1 || !strings.Contains(config.Diagnostics[0].Message, "matched no files") {
		t.Fatalf("got diagnostics %#v, want missing include diagnostic", config.Diagnostics)
	}
}

func TestParseFileExpandedDetectsIncludeCycle(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.conf")
	b := filepath.Join(dir, "b.conf")
	writeConfig(t, a, "Host a\nInclude b.conf\n")
	writeConfig(t, b, "Host b\nInclude a.conf\n")

	config, err := ParseFileExpanded(a)
	if err != nil {
		t.Fatal(err)
	}
	if got := hostPatterns(config); got != "a b" {
		t.Errorf("got hosts %q, want %q", got, "a b")
	}
	if len(config.Diagnostics) != 1 || !strings.Contains(config.Diagnostics[0].Message, "cycle") {
		t.Fatalf("got diagnostics %#v, want cycle diagnostic", config.Diagnostics)
	}
}

func TestParseFileExpandedDoesNotResolveConditionalInclude(t *testing.T) {
	root := filepath.Join(t.TempDir(), "config")
	writeConfig(t, root, "Match host *.example.org\nInclude missing.conf\nHost plain\n")

	config, err := ParseFileExpanded(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Includes) != 1 {
		t.Fatalf("got %d includes, want 1", len(config.Includes))
	}
	if len(config.Diagnostics) != 2 || !strings.Contains(config.Diagnostics[1].Message, "conditional Include") {
		t.Fatalf("got diagnostics %#v, want Match and conditional Include diagnostics", config.Diagnostics)
	}
}

func writeConfig(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func hostPatterns(config *Config) string {
	patterns := make([]string, 0, len(config.Hosts))
	for _, host := range config.Hosts {
		patterns = append(patterns, strings.Join(host.Patterns, " "))
	}
	return strings.Join(patterns, " ")
}
