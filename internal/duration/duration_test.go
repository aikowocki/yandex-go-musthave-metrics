package duration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeconds_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"seconds", `"1s"`, time.Second, false},
		{"ten seconds", `"10s"`, 10 * time.Second, false},
		{"minutes", `"5m"`, 5 * time.Minute, false},
		{"invalid string", `"abc"`, 0, true},
		{"number not string", `10`, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Seconds
			err := s.UnmarshalJSON([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, time.Duration(s))
		})
	}
}

func TestSeconds_UnmarshalText(t *testing.T) {
	var s Seconds
	require.NoError(t, s.UnmarshalText([]byte("42")))
	assert.Equal(t, 42*time.Second, time.Duration(s))
	assert.Equal(t, "42", s.String())
}
