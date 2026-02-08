package main

import (
	"path/filepath"
	"testing"
)

func loadTestConfig(t *testing.T) *Config {
	t.Helper()
	path, _ := filepath.Abs("testdata/valid.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return cfg
}

func TestLoad(t *testing.T) {
	cfg := loadTestConfig(t)

	if cfg.Defaults.Profile != "personal" {
		t.Errorf("defaults.profile = %q, want %q", cfg.Defaults.Profile, "personal")
	}
	if len(cfg.Profiles) != 2 {
		t.Errorf("len(profiles) = %d, want 2", len(cfg.Profiles))
	}
	if len(cfg.Rules) != 2 {
		t.Errorf("len(rules) = %d, want 2", len(cfg.Rules))
	}

	work := cfg.Profiles["work"]
	if work.GHUser != "work-login" {
		t.Errorf("work.gh_user = %q, want %q", work.GHUser, "work-login")
	}
	if work.Git.Name != "Work Name" {
		t.Errorf("work.git.name = %q, want %q", work.Git.Name, "Work Name")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("Load should fail for nonexistent file")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Defaults: Defaults{Profile: "p"},
				Profiles: map[string]Profile{
					"p": {GHUser: "u"},
				},
			},
			wantErr: false,
		},
		{
			name: "defaults.profile not found",
			cfg: Config{
				Defaults: Defaults{Profile: "nonexistent"},
				Profiles: map[string]Profile{
					"p": {GHUser: "u"},
				},
			},
			wantErr: true,
		},
		{
			name: "rule references nonexistent profile",
			cfg: Config{
				Defaults: Defaults{Profile: "p"},
				Profiles: map[string]Profile{
					"p": {GHUser: "u"},
				},
				Rules: []Rule{
					{Hostname: "github.com", Org: "org", Profile: "bad"},
				},
			},
			wantErr: true,
		},
		{
			name: "gh_user not set",
			cfg: Config{
				Defaults: Defaults{Profile: "p"},
				Profiles: map[string]Profile{
					"p": {},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	cfg := loadTestConfig(t)

	tests := []struct {
		name       string
		hostname   string
		org        string
		wantGHUser string
	}{
		{
			name:       "matches rule",
			hostname:   "github.com",
			org:        "work-org",
			wantGHUser: "work-login",
		},
		{
			name:       "matches another work rule",
			hostname:   "github.com",
			org:        "another-work-org",
			wantGHUser: "work-login",
		},
		{
			name:       "falls back to defaults on no match",
			hostname:   "github.com",
			org:        "unknown-org",
			wantGHUser: "personal-login",
		},
		{
			name:       "falls back to defaults on hostname mismatch",
			hostname:   "github.example.com",
			org:        "work-org",
			wantGHUser: "personal-login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := cfg.Resolve(tt.hostname, tt.org)
			if err != nil {
				t.Fatalf("Resolve(%q, %q) error = %v", tt.hostname, tt.org, err)
			}
			if p.GHUser != tt.wantGHUser {
				t.Errorf("Resolve(%q, %q).GHUser = %q, want %q",
					tt.hostname, tt.org, p.GHUser, tt.wantGHUser)
			}
		})
	}
}
