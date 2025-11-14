package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type LoggingConfig struct {
	SQLitePath string `yaml:"sqlite_path"`
}

type YubiKeyCredential struct {
	RPID         string `yaml:"rp_id"`
	CredentialID string `yaml:"credential_id"` // base64url-encoded
	PublicKey    string `yaml:"public_key"`    // base64url-encoded raw public key bytes
}

type AuthConfig struct {
	RequireYubiKey bool   `yaml:"require_yubikey"`
	YubiKeyMode    string `yaml:"yubikey_mode"`
}

type User struct {
	ID           string              `yaml:"id"`
	SSHUsers     []string            `yaml:"ssh_users"`
	Roles        []string            `yaml:"roles"`
	YubiKeyCreds []YubiKeyCredential `yaml:"yubikey_credentials"`
}

type HTTPResource struct {
	BaseURL string `yaml:"base_url"`
}

type PostgresResource struct {
	DSNEnv string `yaml:"dsn_env"`
}

type ResourcesConfig struct {
	HTTP     map[string]HTTPResource     `yaml:"http"`
	Postgres map[string]PostgresResource `yaml:"postgres"`
}

type Operation struct {
	ID           string   `yaml:"id"`
	Label        string   `yaml:"label"`
	Type         string   `yaml:"type"`   // "http" | "postgres"
	Target       string   `yaml:"target"` // key into resources
	Method       string   `yaml:"method"` // for http
	Path         string   `yaml:"path"`   // for http
	Query        string   `yaml:"query"`  // for postgres
	AllowedRoles []string `yaml:"allowed_roles"`
}

type OpenAPIBackend struct {
	DocURL          string   `yaml:"doc_url"`
	TagFilter       []string `yaml:"tag_filter"`
	IncludeUntagged bool     `yaml:"include_untagged"`
	OpIDPrefix      string   `yaml:"op_id_prefix"`
}

type OpenAPIConfig struct {
	Backends map[string]OpenAPIBackend `yaml:"openapi"`
}

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type OnErrorPolicy string

const (
	OnErrorFailFast   OnErrorPolicy = "fail_fast"
	OnErrorBestEffort OnErrorPolicy = "best_effort"
)

type StepOnError string

const (
	StepOnErrorInherit  StepOnError = "inherit"
	StepOnErrorFail     StepOnError = "fail"
	StepOnErrorWarn     StepOnError = "warn"
	StepOnErrorContinue StepOnError = "continue"
)

type TaskStep struct {
	ID       string      `yaml:"id"`
	Type     string      `yaml:"type"`     // "http" | "postgres" | "redis" | "sleep"
	Resource string      `yaml:"resource"` // key in resources.* maps (except sleep)
	Method   string      `yaml:"method"`   // http
	Path     string      `yaml:"path"`     // http
	Query    string      `yaml:"query"`    // postgres
	Command  string      `yaml:"command"`  // redis
	Seconds  int         `yaml:"seconds"`  // sleep
	OnError  StepOnError `yaml:"on_error"`
}

type Task struct {
	ID              string        `yaml:"id"`
	Label           string        `yaml:"label"`
	AllowedRoles    []string      `yaml:"allowed_roles"`
	RiskLevel       RiskLevel     `yaml:"risk_level"`
	RequireYubiKey  bool          `yaml:"require_yubikey"`
	OnError         OnErrorPolicy `yaml:"on_error"`
	Steps           []TaskStep    `yaml:"steps"`
	SummaryTemplate string        `yaml:"summary_template"`
}

type Config struct {
	Project    string          `yaml:"project"`
	Env        string          `yaml:"env"`
	Logging    LoggingConfig   `yaml:"logging"`
	Auth       AuthConfig      `yaml:"auth"`
	Users      []User          `yaml:"users"`
	Resources  ResourcesConfig `yaml:"resources"`
	Operations []Operation     `yaml:"operations"`
	OpenAPI    OpenAPIConfig   `yaml:"openapi"`
	Tasks      []Task          `yaml:"tasks"`
}

func Load() (*Config, error) {
	path := os.Getenv("LAZYADMIN_CONFIG_PATH")
	if path == "" {
		path = "config/lazyadmin.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
