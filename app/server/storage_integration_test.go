package server_test

import (
	"context"
	"github.com/gofiber/storage/redis/v3"
	"github.com/google/uuid"
	"goravel/app/server"
	"os"
	"testing"
	"time"
)

func TestRedisNamespaceIsolation(t *testing.T) {
	if os.Getenv("CHITA_INTEGRATION") != "1" {
		t.Skip("requires local test Redis")
	}
	base := redis.New(redis.Config{Host: "127.0.0.1", Port: 6379, Database: 0})
	defer base.Close()
	store := server.Namespace{Storage: base}
	key := "test:" + uuid.NewString()
	defer store.Delete(key)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := store.SetWithContext(ctx, key, []byte("only-chita"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if data, err := base.Get(key); err != nil || len(data) != 0 {
		t.Fatal("unprefixed key was changed", err)
	}
	if data, err := base.Get("chita:session:" + key); err != nil || string(data) != "only-chita" {
		t.Fatal("namespace missing", err)
	}
	if data, err := store.GetWithContext(ctx, key); err != nil || string(data) != "only-chita" {
		t.Fatal("context read failed", err)
	}
	if store.Reset() == nil || store.ResetWithContext(ctx) == nil {
		t.Fatal("global reset must be disabled")
	}
	if err := store.DeleteWithContext(ctx, key); err != nil {
		t.Fatal(err)
	}
	if data, err := store.Get(key); err != nil || len(data) != 0 {
		t.Fatal("key was not deleted", err)
	}
}
