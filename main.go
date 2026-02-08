package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "ghmux", "config.yaml"), nil
}

func loadDefaultConfig() (*Config, error) {
	path, err := defaultConfigPath()
	if err != nil {
		return nil, err
	}
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func runCmd(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// parseRemoteURL extracts hostname and org from a git remote URL.
func parseRemoteURL(rawURL string) (hostname, org string) {
	// SCP format: git@github.com:org/repo.git
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") && !strings.HasPrefix(rawURL, "ssh://") {
		parts := strings.SplitN(rawURL, ":", 2)
		hostPart := strings.SplitN(parts[0], "@", 2)
		hostname = hostPart[len(hostPart)-1]
		pathPart := strings.TrimSuffix(parts[1], ".git")
		segs := strings.SplitN(pathPart, "/", 2)
		if len(segs) >= 1 {
			org = segs[0]
		}
		return hostname, org
	}

	// HTTPS / ssh:// format
	u := strings.TrimSuffix(rawURL, ".git")
	u = strings.TrimPrefix(u, "ssh://git@")
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	segs := strings.SplitN(u, "/", 3)
	if len(segs) >= 2 {
		return segs[0], segs[1]
	}
	return "", ""
}

func getRemoteOrg() (hostname, org string, err error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get git remote (run this inside a git repository)")
	}
	hostname, org = parseRemoteURL(strings.TrimSpace(string(out)))
	if hostname == "" || org == "" {
		return "", "", fmt.Errorf("could not determine org from remote URL: %s", strings.TrimSpace(string(out)))
	}
	return hostname, org, nil
}

const managedHeader = "# >>> ghmux managed (DO NOT EDIT) >>>"
const managedFooter = "# <<< ghmux managed <<<"

func syncGitconfig(cfg *Config, homeDir string) error {
	home := homeDir

	for name, p := range cfg.Profiles {
		if p.Git.Name == "" && p.Git.Email == "" {
			continue
		}
		path := filepath.Join(home, fmt.Sprintf(".gitconfig-%s", name))
		content := fmt.Sprintf("[user]\n\tname = %s\n\temail = %s\n", p.Git.Name, p.Git.Email)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
		fmt.Fprintf(os.Stderr, "generated: %s\n", path)
	}

	block := buildIncludeIfBlock(cfg)

	gitconfigPath := filepath.Join(home, ".gitconfig")
	existing, _ := os.ReadFile(gitconfigPath)
	updated := replaceManaged(string(existing), block)
	if err := os.WriteFile(gitconfigPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write .gitconfig: %w", err)
	}
	fmt.Fprintf(os.Stderr, "updated: %s\n", gitconfigPath)
	return nil
}

func buildIncludeIfBlock(cfg *Config) string {
	var block []string
	for _, rule := range cfg.Rules {
		if rule.Hostname == "" {
			continue
		}
		path := fmt.Sprintf("\tpath = ~/.gitconfig-%s", rule.Profile)

		https := fmt.Sprintf(`[includeIf "hasconfig:remote.*.url:https://%s/%s/**"]`, rule.Hostname, rule.Org)
		block = append(block, https, path)

		scp := fmt.Sprintf(`[includeIf "hasconfig:remote.*.url:git@%s:%s/**"]`, rule.Hostname, rule.Org)
		block = append(block, scp, path)
	}

	if len(block) == 0 {
		return ""
	}
	return strings.Join(block, "\n") + "\n"
}

// replaceManaged replaces the managed block in an existing gitconfig.
func replaceManaged(existing, block string) string {
	managed := managedHeader + "\n" + block + managedFooter + "\n"

	startIdx := strings.Index(existing, managedHeader)
	endIdx := strings.Index(existing, managedFooter)
	if startIdx >= 0 && endIdx >= 0 {
		return existing[:startIdx] + managed + existing[endIdx+len(managedFooter)+1:]
	}

	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	return existing + managed
}

func main() {
	root := &cobra.Command{
		Use:   "ghmux",
		Short: "CLI for managing multiple GitHub accounts",
	}

	root.AddCommand(newSyncCmd())
	root.AddCommand(newInitCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Auto-switch gh account based on CWD remote",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadDefaultConfig()
			if err != nil {
				return err
			}
			hostname, org, err := getRemoteOrg()
			if err != nil {
				return err
			}
			profile, err := cfg.Resolve(hostname, org)
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "gh auth switch → %s (org: %s)\n", profile.GHUser, org)
			return runCmd("gh", []string{"auth", "switch", "--hostname", hostname, "--user", profile.GHUser})
		},
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Generate includeIf and per-profile gitconfig (initial setup)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadDefaultConfig()
			if err != nil {
				return err
			}
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			return syncGitconfig(cfg, home)
		},
	}
}
