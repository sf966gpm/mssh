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
