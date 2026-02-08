package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantHostname string
		wantOrg      string
	}{
		{
			name:         "HTTPS format",
			url:          "https://github.com/myorg/myrepo.git",
			wantHostname: "github.com",
			wantOrg:      "myorg",
		},
		{
			name:         "HTTPS without .git",
			url:          "https://github.com/myorg/myrepo",
			wantHostname: "github.com",
			wantOrg:      "myorg",
		},
		{
			name:         "SCP format",
			url:          "git@github.com:myorg/myrepo.git",
			wantHostname: "github.com",
			wantOrg:      "myorg",
		},
		{
			name:         "ssh:// format",
			url:          "ssh://git@github.com/myorg/myrepo.git",
			wantHostname: "github.com",
			wantOrg:      "myorg",
		},
		{
			name:         "GitHub Enterprise",
			url:          "https://github.example.com/internal/project.git",
			wantHostname: "github.example.com",
			wantOrg:      "internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostname, org := parseRemoteURL(tt.url)
			if hostname != tt.wantHostname {
				t.Errorf("hostname = %q, want %q", hostname, tt.wantHostname)
			}
			if org != tt.wantOrg {
				t.Errorf("org = %q, want %q", org, tt.wantOrg)
			}
		})
	}
}

func TestBuildIncludeIfBlock(t *testing.T) {
	cfg := &Config{
		Rules: []Rule{
			{Hostname: "github.com", Org: "work-org", Profile: "work"},
			{Hostname: "github.example.com", Org: "internal", Profile: "enterprise"},
		},
	}

	got := buildIncludeIfBlock(cfg)

	// Verify both HTTPS and SCP patterns are present for each rule
	expectations := []struct {
		desc    string
		pattern string
	}{
		{"HTTPS work-org", `[includeIf "hasconfig:remote.*.url:https://github.com/work-org/**"]`},
		{"SCP work-org", `[includeIf "hasconfig:remote.*.url:git@github.com:work-org/**"]`},
		{"path work", "path = ~/.gitconfig-work"},
		{"HTTPS internal", `[includeIf "hasconfig:remote.*.url:https://github.example.com/internal/**"]`},
		{"SCP internal", `[includeIf "hasconfig:remote.*.url:git@github.example.com:internal/**"]`},
		{"path enterprise", "path = ~/.gitconfig-enterprise"},
	}

	for _, e := range expectations {
		if !strings.Contains(got, e.pattern) {
			t.Errorf("%s: %q not found in output\ngot:\n%s", e.desc, e.pattern, got)
		}
	}
}

func TestBuildIncludeIfBlock_EmptyRules(t *testing.T) {
	cfg := &Config{Rules: []Rule{}}
	got := buildIncludeIfBlock(cfg)
	if got != "" {
		t.Errorf("expected empty string for empty rules, got %q", got)
	}
}

func TestBuildIncludeIfBlock_EmptyHostname(t *testing.T) {
	cfg := &Config{
		Rules: []Rule{
			{Hostname: "", Org: "my-org", Profile: "work"},
		},
	}
	got := buildIncludeIfBlock(cfg)

	// Rules with empty hostname should be skipped
	if got != "" {
		t.Errorf("rules with empty hostname should be skipped, but got output:\n%s", got)
	}
}

func TestSyncGitconfig(t *testing.T) {
	home := t.TempDir()
	cfg := &Config{
		Profiles: map[string]Profile{
			"work": {
				GHUser: "work-login",
				Git:    GitAuthor{Name: "Work Name", Email: "work@example.com"},
			},
			"personal": {
				GHUser: "personal-login",
			},
		},
		Rules: []Rule{
			{Hostname: "github.com", Org: "work-org", Profile: "work"},
		},
	}

	if err := syncGitconfig(cfg, home); err != nil {
		t.Fatalf("syncGitconfig() error = %v", err)
	}

	// Per-profile gitconfig should be generated
	workConfig, err := os.ReadFile(filepath.Join(home, ".gitconfig-work"))
	if err != nil {
		t.Fatalf(".gitconfig-work was not generated: %v", err)
	}
	if !strings.Contains(string(workConfig), "Work Name") {
		t.Errorf(".gitconfig-work does not contain name: %s", workConfig)
	}
	if !strings.Contains(string(workConfig), "work@example.com") {
		t.Errorf(".gitconfig-work does not contain email: %s", workConfig)
	}

	// personal with empty git.name/email should not be generated
	if _, err := os.Stat(filepath.Join(home, ".gitconfig-personal")); err == nil {
		t.Error(".gitconfig-personal should not be generated when git config is empty")
	}

	// .gitconfig should contain the managed block
	gitconfig, err := os.ReadFile(filepath.Join(home, ".gitconfig"))
	if err != nil {
		t.Fatalf(".gitconfig was not generated: %v", err)
	}
	content := string(gitconfig)
	if !strings.Contains(content, managedHeader) {
		t.Error(".gitconfig does not contain managed header")
	}
	if !strings.Contains(content, managedFooter) {
		t.Error(".gitconfig does not contain managed footer")
	}
	if !strings.Contains(content, "includeIf") {
		t.Error(".gitconfig does not contain includeIf block")
	}
}

func TestSyncGitconfig_Idempotent(t *testing.T) {
	home := t.TempDir()
	cfg := &Config{
		Profiles: map[string]Profile{
			"work": {
				GHUser: "work-login",
				Git:    GitAuthor{Name: "Work Name", Email: "work@example.com"},
			},
		},
		Rules: []Rule{
			{Hostname: "github.com", Org: "work-org", Profile: "work"},
		},
	}

	// Run with existing .gitconfig
	existing := "[alias]\n\tco = checkout\n"
	os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(existing), 0644)

	if err := syncGitconfig(cfg, home); err != nil {
		t.Fatalf("first run: %v", err)
	}
	first, _ := os.ReadFile(filepath.Join(home, ".gitconfig"))

	// Content should not change on second run
	if err := syncGitconfig(cfg, home); err != nil {
		t.Fatalf("second run: %v", err)
	}
	second, _ := os.ReadFile(filepath.Join(home, ".gitconfig"))

	if string(first) != string(second) {
		t.Errorf("idempotency broken:\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	// Existing config should be preserved
	if !strings.Contains(string(second), "[alias]") {
		t.Error("existing [alias] section was lost")
	}
}

func TestReplaceManaged(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		block    string
		want     string
	}{
		{
			name:     "append to empty file",
			existing: "",
			block:    "[includeIf]\n\tpath = test\n",
			want:     "# >>> ghmux managed (DO NOT EDIT) >>>\n[includeIf]\n\tpath = test\n# <<< ghmux managed <<<\n",
		},
		{
			name:     "append after existing config",
			existing: "[user]\n\tname = Test\n",
			block:    "[includeIf]\n\tpath = test\n",
			want:     "[user]\n\tname = Test\n# >>> ghmux managed (DO NOT EDIT) >>>\n[includeIf]\n\tpath = test\n# <<< ghmux managed <<<\n",
		},
		{
			name:     "replace existing managed block",
			existing: "[user]\n\tname = Test\n# >>> ghmux managed (DO NOT EDIT) >>>\nold block\n# <<< ghmux managed <<<\n[alias]\n\tco = checkout\n",
			block:    "new block\n",
			want:     "[user]\n\tname = Test\n# >>> ghmux managed (DO NOT EDIT) >>>\nnew block\n# <<< ghmux managed <<<\n[alias]\n\tco = checkout\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceManaged(tt.existing, tt.block)
			if got != tt.want {
				t.Errorf("replaceManaged():\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
