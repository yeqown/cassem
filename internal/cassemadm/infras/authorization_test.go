package infras

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSessionRejectsLegacyUnsignedToken(t *testing.T) {
	legacy, err := json.Marshal(&Session{
		Account:   "superadmin",
		Salt:      "known-salt",
		ExpiredAt: time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)

	_, err = parseSession(base64.StdEncoding.EncodeToString(legacy), "test-secret")
	require.Error(t, err)
}

func TestSessionRejectsTamperedPayload(t *testing.T) {
	sess := &Session{
		Account:   "alice",
		Salt:      "alice-salt",
		ExpiredAt: time.Now().Add(time.Hour).Unix(),
	}
	token, err := EncodeSession(sess, "test-secret")
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	require.Len(t, parts, 2)

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)

	var tampered Session
	require.NoError(t, json.Unmarshal(payload, &tampered))
	tampered.Account = "superadmin"
	payload, err = json.Marshal(&tampered)
	require.NoError(t, err)

	_, err = parseSession(base64.RawURLEncoding.EncodeToString(payload)+"."+parts[1], "test-secret")
	require.Error(t, err)
}

func TestSessionParsesSignedToken(t *testing.T) {
	want := &Session{
		Account:   "alice",
		Salt:      "alice-salt",
		ExpiredAt: time.Now().Add(time.Hour).Unix(),
	}
	token, err := EncodeSession(want, "test-secret")
	require.NoError(t, err)

	got, err := parseSession(token, "test-secret")
	require.NoError(t, err)
	require.Equal(t, want.Account, got.Account)
	require.Equal(t, want.Salt, got.Salt)
	require.Equal(t, want.ExpiredAt, got.ExpiredAt)
}
