//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeKiroCredentials_AuthMethodIdCCustom(t *testing.T) {
	creds := map[string]any{
		"refresh_token": "rt",
		"auth_method":   "idc_custom",
	}

	got := NormalizeKiroCredentials(creds)

	require.Equal(t, "IdC", got["auth_type"])
	_, hasAuthMethod := got["auth_method"]
	require.False(t, hasAuthMethod)
}

func TestNormalizeKiroCredentials_ProviderBuilderIDAutoDetectsIdC(t *testing.T) {
	creds := map[string]any{
		"refresh_token": "rt",
		"provider":      "builderid",
	}

	got := NormalizeKiroCredentials(creds)

	require.Equal(t, "IdC", got["auth_type"])
}

func TestNormalizeKiroCredentials_AuthTypeAliasBuilderID(t *testing.T) {
	creds := map[string]any{
		"refresh_token": "rt",
		"auth_type":     "builder_id",
	}

	got := NormalizeKiroCredentials(creds)

	require.Equal(t, "IdC", got["auth_type"])
}

