package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueToken_ContainsUserID(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", "invest-guide", time.Hour)
	token, err := issuer.Issue("user-123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := issuer.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "invest-guide", claims.Issuer)
}

func TestVerify_RejectsWrongSecret(t *testing.T) {
	issuerA := NewJWTIssuer("secret-a", "invest-guide", time.Hour)
	issuerB := NewJWTIssuer("secret-b", "invest-guide", time.Hour)

	token, _ := issuerA.Issue("user-1")
	_, err := issuerB.Verify(token)
	assert.Error(t, err)
}

func TestVerify_RejectsExpiredToken(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", "invest-guide", -time.Hour)
	token, _ := issuer.Issue("user-1")
	_, err := issuer.Verify(token)
	assert.Error(t, err)
}

func TestVerify_RejectsWrongIssuer(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", "invest-guide", time.Hour)
	token, _ := issuer.Issue("user-1")

	other := NewJWTIssuer("test-secret", "other-issuer", time.Hour)
	_, err := other.Verify(token)
	assert.Error(t, err)
}
