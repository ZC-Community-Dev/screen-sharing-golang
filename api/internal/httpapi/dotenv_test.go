package httpapi

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReadsSaltFromEnvFileWithoutLogging(t *testing.T) {
	unsetForTest(t, "LINK_ID_SALT")
	unsetForTest(t, "LINKS_DB_PATH")
	unsetForTest(t, "PORT")

	path := filepath.Join(t.TempDir(), ".env")
	const salt = "dotenv-file-salt-xyz"
	if err := os.WriteFile(path, []byte("LINK_ID_SALT="+salt+"\nPORT=8099\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	if err := LoadEnvFile(path); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Salt != salt {
		t.Fatalf("salt %q", cfg.Salt)
	}
	if cfg.Port != "8099" {
		t.Fatalf("port %q", cfg.Port)
	}
	logger.Info("config loaded", "ok", true)
	if strings.Contains(buf.String(), salt) {
		t.Fatal("salt appeared in logs")
	}
}

func TestOSEnvWinsOverEnvFile(t *testing.T) {
	t.Setenv("LINK_ID_SALT", "from-os")
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("LINK_ID_SALT=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LoadEnvFile(path); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Salt != "from-os" {
		t.Fatalf("got %q, want OS value", cfg.Salt)
	}
}

func unsetForTest(t *testing.T, key string) {
	t.Helper()
	orig, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, orig)
			return
		}
		_ = os.Unsetenv(key)
	})
}
