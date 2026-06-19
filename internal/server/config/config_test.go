package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTempConfig записывает content во временный JSON-файл и возвращает его путь.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestParseServerConfig_Defaults(t *testing.T) {
	cfg, err := parseServerConfig(nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "localhost:8080", cfg.ServerAddress)
	assert.Equal(t, 300*time.Second, time.Duration(cfg.StoreInterval))
	assert.True(t, cfg.Restore)
	assert.Equal(t, "backup/metrics.json", cfg.FileStoragePath)
	assert.Equal(t, "localhost:6060", cfg.PprofAddress)
}

func TestParseServerConfig_JSONFile(t *testing.T) {
	path := writeTempConfig(t, `{
		"address": "json-addr:9090",
		"restore": false,
		"store_interval": "7s",
		"store_file": "/json/store.db",
		"database_dsn": "postgres://json-dsn",
		"key": "json-key",
		"crypto_key": "/json/key.pem",
		"audit_file": "/json/audit.log",
		"audit_url": "http://json/audit",
		"pprof_address": "localhost:7777"
	}`)

	cfg, err := parseServerConfig([]string{"-c", path}, nil)
	require.NoError(t, err)

	assert.Equal(t, "json-addr:9090", cfg.ServerAddress)
	assert.False(t, cfg.Restore)
	assert.Equal(t, 7*time.Second, time.Duration(cfg.StoreInterval))
	assert.Equal(t, "/json/store.db", cfg.FileStoragePath)
	assert.Equal(t, "postgres://json-dsn", cfg.DB.DatabaseDSN, "database_dsn → DB.DatabaseDSN")
	assert.Equal(t, "json-key", cfg.Key)
	assert.Equal(t, "/json/key.pem", cfg.CryptoKey)
	assert.Equal(t, "/json/audit.log", cfg.AuditFile)
	assert.Equal(t, "http://json/audit", cfg.AuditURL)
	assert.Equal(t, "localhost:7777", cfg.PprofAddress)
}

// TestParseServerConfig_PartialJSON проверяет сохранение дефолтов
// для полей, отсутствующих в JSON.
func TestParseServerConfig_PartialJSON(t *testing.T) {
	path := writeTempConfig(t, `{"address": "only-addr:1234"}`)

	cfg, err := parseServerConfig([]string{"-c", path}, nil)
	require.NoError(t, err)

	assert.Equal(t, "only-addr:1234", cfg.ServerAddress)
	assert.True(t, cfg.Restore, "default true preserved")
	assert.Equal(t, 300*time.Second, time.Duration(cfg.StoreInterval), "default preserved")
	assert.Equal(t, "backup/metrics.json", cfg.FileStoragePath, "default preserved")
}

// TestParseServerConfig_RestoreFalse проверяет, что явный false в JSON
// не затирается дефолтом true (zero-value ловушка для bool).
func TestParseServerConfig_RestoreFalse(t *testing.T) {
	path := writeTempConfig(t, `{"restore": false}`)

	cfg, err := parseServerConfig([]string{"-c", path}, nil)
	require.NoError(t, err)

	assert.False(t, cfg.Restore)
}

// TestParseServerConfig_AllLayers проверяет все приоритеты в одном тесте:
// env > flag > JSON > default.
func TestParseServerConfig_AllLayers(t *testing.T) {
	path := writeTempConfig(t, `{
		"address": "json-addr:9090",
		"store_interval": "100s",
		"database_dsn": "json-dsn",
		"key": "json-key"
	}`)
	environ := map[string]string{
		"ADDRESS":      "env-addr:2222",
		"DATABASE_DSN": "env-dsn",
	}

	cfg, err := parseServerConfig([]string{"-c", path, "-i", "22"}, environ)
	require.NoError(t, err)

	assert.Equal(t, "env-addr:2222", cfg.ServerAddress, "env > flag/JSON")
	assert.Equal(t, "env-dsn", cfg.DB.DatabaseDSN, "env > JSON")
	assert.Equal(t, 22*time.Second, time.Duration(cfg.StoreInterval), "flag > JSON")
	assert.Equal(t, "json-key", cfg.Key, "JSON применяется когда нет flag/env")
	assert.Equal(t, "localhost:6060", cfg.PprofAddress, "дефолт когда нигде не задано")
}

func TestParseServerConfig_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"bad json", []string{"-c", writeTempConfig(t, `{broken`)}},
		{"missing file", []string{"-c", "/no/such/file.json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseServerConfig(tt.args, nil)
			assert.Error(t, err)
		})
	}
}
