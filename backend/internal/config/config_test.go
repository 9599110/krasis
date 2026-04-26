package config

import (
	"os"
	"path/filepath"
	"testing"
)

// M1.1 验收：YAML 配置，支持 CONFIG_FILE 环境变量覆盖（见 docs/验收标准.md 1.1）
func TestLoad_CONFIG_FILEOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yaml")
	if err := os.WriteFile(path, []byte(`
server:
  port: 9999
  mode: release
jwt:
  secret: from-file
  issuer: test-issuer
  expiration: 3600
`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CONFIG_FILE", path)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9999 || cfg.Server.Mode != "release" {
		t.Fatalf("server: %+v", cfg.Server)
	}
	if cfg.JWT.Secret != "from-file" || cfg.JWT.Issuer != "test-issuer" || cfg.JWT.Expiration != 3600 {
		t.Fatalf("jwt: %+v", cfg.JWT)
	}
}

func TestLoad_DefaultsWhenNoFile(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if prev, ok := os.LookupEnv("CONFIG_FILE"); ok {
		_ = os.Unsetenv("CONFIG_FILE")
		t.Cleanup(func() { _ = os.Setenv("CONFIG_FILE", prev) })
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("default port: %d", cfg.Server.Port)
	}
	if cfg.Log.Format != "json" {
		t.Fatalf("default log.format: %q", cfg.Log.Format)
	}
}
