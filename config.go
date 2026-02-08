package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type GitAuthor struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type Profile struct {
	GHUser string    `yaml:"gh_user"`
	Git    GitAuthor `yaml:"git"`
}

type Rule struct {
	Hostname string `yaml:"hostname"`
	Org      string `yaml:"org"`
	Profile  string `yaml:"profile"`
}

type Defaults struct {
	Profile string `yaml:"profile"`
}

type Config struct {
	Defaults Defaults           `yaml:"defaults"`
	Profiles map[string]Profile `yaml:"profiles"`
	Rules    []Rule             `yaml:"rules"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if _, ok := c.Profiles[c.Defaults.Profile]; !ok {
		return fmt.Errorf("defaults.profile %q not found in profiles", c.Defaults.Profile)
	}
	for _, p := range c.Profiles {
		if p.GHUser == "" {
			return fmt.Errorf("gh_user is not set in profile")
		}
	}
	for i, r := range c.Rules {
		if _, ok := c.Profiles[r.Profile]; !ok {
			return fmt.Errorf("rules[%d].profile %q not found in profiles", i, r.Profile)
		}
	}
	return nil
}

// Resolve returns the matching profile for the given hostname and org.
// It evaluates rules top-down and returns the first match; falls back to defaults if none match.
func (c *Config) Resolve(hostname, org string) (*Profile, error) {

	var profile Profile
	for _, rule := range c.Rules {
		if rule.Org != org {
			continue
		}
		if rule.Hostname == hostname || rule.Hostname == "" {
			profile = c.Profiles[rule.Profile]
			return &profile, nil
		}
	}

	profile = c.Profiles[c.Defaults.Profile]
	return &profile, nil
}
