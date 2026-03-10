package cache

import (
	"testing"
	"time"

	"github.com/yogayulanda/go-core/config"
)

func TestNewRedisFromConfig_InvalidAddress_ReturnError(t *testing.T) {
	_, err := NewRedisFromConfig(config.RedisConfig{
		Enabled: true,
		Address: "127.0.0.1:0",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewMemcachedFromConfig_InvalidAddress_ReturnError(t *testing.T) {
	_, err := NewMemcachedFromConfig(config.MemcachedConfig{
		Enabled: true,
		Servers: []string{"127.0.0.1:1"},
		Timeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
