package audit

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileObserver_WritesJSONLines(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "audit-*.jsonl")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	obs, err := NewFileObserver(f.Name())
	require.NoError(t, err)
	defer func() { _ = obs.Close() }()

	events := []entity.AuditEvent{
		{Timestamp: 100, Metrics: []string{"Alloc"}, IPAddress: "10.0.0.1"},
		{Timestamp: 200, Metrics: []string{"Frees", "GCSys"}, IPAddress: "10.0.0.2"},
	}

	for _, e := range events {
		require.NoError(t, obs.Notify(t.Context(), e))
	}

	data, err := os.ReadFile(f.Name())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 2)

	for i, line := range lines {
		var got entity.AuditEvent
		require.NoError(t, json.Unmarshal([]byte(line), &got))
		assert.Equal(t, events[i], got)
	}
}

func TestFileObserver_InvalidPath(t *testing.T) {
	_, err := NewFileObserver("/nonexistent/dir/file.jsonl")
	assert.Error(t, err)
}
