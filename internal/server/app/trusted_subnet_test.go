package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTrustedSubnet(t *testing.T) {
	t.Run("empty disables filtering", func(t *testing.T) {
		subnet, err := parseTrustedSubnet("")
		require.NoError(t, err)
		assert.Nil(t, subnet)
	})

	t.Run("valid cidr parsed", func(t *testing.T) {
		subnet, err := parseTrustedSubnet("172.28.0.0/16")
		require.NoError(t, err)
		require.NotNil(t, subnet)
		assert.Equal(t, "172.28.0.0/16", subnet.String())
	})

	t.Run("invalid cidr fails fast", func(t *testing.T) {
		subnet, err := parseTrustedSubnet("172.28.0.0/333")
		require.Error(t, err)
		assert.Nil(t, subnet)
	})

	t.Run("plain ip without mask fails", func(t *testing.T) {
		subnet, err := parseTrustedSubnet("10.0.0.1")
		require.Error(t, err)
		assert.Nil(t, subnet)
	})
}
