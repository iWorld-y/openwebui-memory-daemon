package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	OpenWebUI struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
	} `yaml:"openwebui"`

	LLM struct {
		BaseURL    string `yaml:"base_url"`
		APIKey     string `yaml:"api_key"`
		Model      string `yaml:"model"`
		MaxTokens  int    `yaml:"max_tokens"`
		TimeoutSec int    `yaml:"timeout_sec"`
	} `yaml:"llm"`

	Schedule struct {
		Daily   string `yaml:"daily"`
		Weekly  string `yaml:"weekly"`
		Monthly string `yaml:"monthly"`
	} `yaml:"schedule"`

	Git struct {
		RepoPath    string `yaml:"repo_path"`
		Remote      string `yaml:"remote"`
		Branch      string `yaml:"branch"`
		AuthorName  string `yaml:"author_name"`
		AuthorEmail string `yaml:"author_email"`
	} `yaml:"git"`

	Log struct {
		Level string `yaml:"level"`
		Path  string `yaml:"path"`
	} `yaml:"log"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
