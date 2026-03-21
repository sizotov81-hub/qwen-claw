package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitSecurity(t *testing.T) {
	t.Run("create new secrets", func(t *testing.T) {
		config := &ServerConfig{}
		err := InitSecurity("/tmp/test_web_secrets", config)

		assert.NoError(t, err)
		assert.NotEmpty(t, config.SecretPhrase)
		assert.NotEmpty(t, config.JWTSecret)
	})

	t.Run("load existing secrets", func(t *testing.T) {
		config := &ServerConfig{}
		_ = InitSecurity("/tmp/test_web_secrets", config)

		originalPhrase := config.SecretPhrase
		originalSecret := config.JWTSecret

		config2 := &ServerConfig{}
		_ = InitSecurity("/tmp/test_web_secrets", config2)

		assert.Equal(t, originalPhrase, config2.SecretPhrase)
		assert.Equal(t, originalSecret, config2.JWTSecret)
	})
}

func TestGenerateSecureToken(t *testing.T) {
	token1 := generateSecureToken(32)
	token2 := generateSecureToken(32)
	token3 := generateSecureToken(64)

	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, token2)
	assert.NotEmpty(t, token3)
	assert.NotEqual(t, token1, token2)
	assert.Len(t, token1, 64)
	assert.Len(t, token3, 128)
}

func TestNewServer(t *testing.T) {
	server := NewServer(
		ServerConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
		nil, nil, nil,
	)

	assert.NotNil(t, server)
	assert.Equal(t, "127.0.0.1", server.config.Host)
	assert.Equal(t, 8080, server.config.Port)
}

func TestAuthClaims(t *testing.T) {
	claims := &AuthClaims{
		Authenticated: true,
		IP:            "127.0.0.1",
	}

	assert.True(t, claims.Authenticated)
	assert.Equal(t, "127.0.0.1", claims.IP)
}

func TestServerConfig(t *testing.T) {
	config := ServerConfig{
		Host:        "0.0.0.0",
		Port:        64656,
		SecretPhrase: "test_secret",
		JWTSecret:   "test_jwt",
	}

	assert.Equal(t, "0.0.0.0", config.Host)
	assert.Equal(t, 64656, config.Port)
	assert.Equal(t, "test_secret", config.SecretPhrase)
	assert.Equal(t, "test_jwt", config.JWTSecret)
}

func TestServerStop(t *testing.T) {
	server := NewServer(
		ServerConfig{
			Host: "127.0.0.1",
			Port: 18080,
		},
		nil, nil, nil,
	)

	assert.NotPanics(t, func() {
		err := server.Stop()
		assert.NoError(t, err)
	})
}

func TestAPIResponse(t *testing.T) {
	response := APIResponse{
		Success: true,
		Data:    map[string]string{"key": "value"},
	}

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
}

func TestChatRequest(t *testing.T) {
	req := ChatRequest{
		Message: "test message",
	}

	assert.Equal(t, "test message", req.Message)
}

func TestChatResponse(t *testing.T) {
	resp := ChatResponse{
		Message:   "test response",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	assert.Equal(t, "test response", resp.Message)
	assert.NotEmpty(t, resp.Timestamp)
}
