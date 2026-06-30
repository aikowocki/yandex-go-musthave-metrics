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

func TestParseAgentConfig_Defaults(t *testing.T) {
	cfg, err := parseAgentConfig(nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "localhost:8080", cfg.ServerAddress)
	assert.Equal(t, 10*time.Second, time.Duration(cfg.ReportInterval))
	assert.Equal(t, 2*time.Second, time.Duration(cfg.PollInterval))
	assert.Equal(t, 1, cfg.RateLimit)
	assert.Equal(t, "localhost:6061", cfg.PprofAddress)
	assert.Empty(t, cfg.GRPCAddress)
}

func TestParseAgentConfig_JSONFile(t *testing.T) {
	path := writeTempConfig(t, `{
		"address": "json-addr:9090",
		"report_interval": "5s",
		"poll_interval": "3s",
		"key": "json-key",
		"crypto_key": "/json/key.pem",
		"rate_limit": 7,
		"pprof_address": "localhost:7777",
		"grpc_address": "localhost:3200"
	}`)

	cfg, err := parseAgentConfig([]string{"-c", path}, nil)
	require.NoError(t, err)

	assert.Equal(t, "json-addr:9090", cfg.ServerAddress)
	assert.Equal(t, 5*time.Second, time.Duration(cfg.ReportInterval))
	assert.Equal(t, 3*time.Second, time.Duration(cfg.PollInterval))
	assert.Equal(t, "json-key", cfg.Key)
	assert.Equal(t, "/json/key.pem", cfg.CryptoKey)
	assert.Equal(t, 7, cfg.RateLimit)
	assert.Equal(t, "localhost:7777", cfg.PprofAddress)
	assert.Equal(t, "localhost:3200", cfg.GRPCAddress)
}

// TestParseAgentConfig_PartialJSON проверяет, что поля, отсутствующие в JSON,
// сохраняют дефолтные значения (json.Unmarshal не затирает неуказанные поля).
func TestParseAgentConfig_PartialJSON(t *testing.T) {
	path := writeTempConfig(t, `{"address": "only-addr:1234"}`)

	cfg, err := parseAgentConfig([]string{"-c", path}, nil)
	require.NoError(t, err)

	assert.Equal(t, "only-addr:1234", cfg.ServerAddress)
	assert.Equal(t, 10*time.Second, time.Duration(cfg.ReportInterval), "default preserved")
	assert.Equal(t, 2*time.Second, time.Duration(cfg.PollInterval), "default preserved")
	assert.Equal(t, 1, cfg.RateLimit, "default preserved")
}

// TestParseAgentConfig_RateLimitZero проверяет, что явный 0 в JSON применяется
// (json.Unmarshal различает "ключа нет" и "ключ со значением 0").
func TestParseAgentConfig_RateLimitZero(t *testing.T) {
	path := writeTempConfig(t, `{"rate_limit": 0}`)

	cfg, err := parseAgentConfig([]string{"-c", path}, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, cfg.RateLimit)
}

// TestParseAgentConfig_AllLayers проверяет все приоритеты в одном тесте:
// env > flag > JSON > default.
func TestParseAgentConfig_AllLayers(t *testing.T) {
	path := writeTempConfig(t, `{
		"address": "json-addr:9090",
		"report_interval": "100s",
		"poll_interval": "100s",
		"key": "json-key"
	}`)
	environ := map[string]string{
		"ADDRESS":         "env-addr:2222",
		"REPORT_INTERVAL": "11",
		"GRPC_ADDRESS":    "env-grpc:4200",
	}

	cfg, err := parseAgentConfig([]string{"-c", path, "-p", "22", "-grpc-address", "flag-grpc:3200"}, environ)
	require.NoError(t, err)

	assert.Equal(t, "env-addr:2222", cfg.ServerAddress, "env > flag/JSON")
	assert.Equal(t, 11*time.Second, time.Duration(cfg.ReportInterval), "env > JSON")
	assert.Equal(t, 22*time.Second, time.Duration(cfg.PollInterval), "flag > JSON")
	assert.Equal(t, "env-grpc:4200", cfg.GRPCAddress, "env > flag")
	assert.Equal(t, "json-key", cfg.Key, "JSON применяется когда нет flag/env")
	assert.Equal(t, "localhost:6061", cfg.PprofAddress, "дефолт когда нигде не задано")
}

// TestParseAgentConfig_EnvConfigPath: env CONFIG имеет приоритет над флагом -c
// при выборе файла конфигурации.
func TestParseAgentConfig_EnvConfigPath(t *testing.T) {
	envPath := writeTempConfig(t, `{"address": "env-file-addr:6666"}`)
	flagPath := writeTempConfig(t, `{"address": "flag-file-addr:7777"}`)
	environ := map[string]string{"CONFIG": envPath}

	cfg, err := parseAgentConfig([]string{"-c", flagPath}, environ)
	require.NoError(t, err)

	assert.Equal(t, "env-file-addr:6666", cfg.ServerAddress, "env CONFIG  > флаг -c")
}

func TestParseAgentConfig_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"bad json", []string{"-c", writeTempConfig(t, `{not a json`)}},
		{"missing file", []string{"-c", "/no/such/file.json"}},
		{"bad duration", []string{"-c", writeTempConfig(t, `{"report_interval": "not-a-duration"}`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAgentConfig(tt.args, nil)
			assert.Error(t, err)
		})
	}
}
