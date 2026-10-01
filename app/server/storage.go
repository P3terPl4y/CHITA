package server

import (
	"context"
	"errors"
	"github.com/gofiber/fiber/v3"
	"time"
)

// Namespace prevents CHITA sessions from colliding with another application.
// Reset deliberately cannot flush a shared Redis database.
type Namespace struct{ fiber.Storage }

func (s Namespace) Get(k string) ([]byte, error) { return s.Storage.Get("chita:session:" + k) }
func (s Namespace) Set(k string, v []byte, d time.Duration) error {
	return s.Storage.Set("chita:session:"+k, v, d)
}
func (s Namespace) Delete(k string) error { return s.Storage.Delete("chita:session:" + k) }
func (s Namespace) GetWithContext(c context.Context, k string) ([]byte, error) {
	return s.Storage.GetWithContext(c, "chita:session:"+k)
}
func (s Namespace) SetWithContext(c context.Context, k string, v []byte, d time.Duration) error {
	return s.Storage.SetWithContext(c, "chita:session:"+k, v, d)
}
func (s Namespace) DeleteWithContext(c context.Context, k string) error {
	return s.Storage.DeleteWithContext(c, "chita:session:"+k)
}
func (s Namespace) Reset() error                           { return errors.New("shared storage reset is disabled") }
func (s Namespace) ResetWithContext(context.Context) error { return s.Reset() }

func sessionStorage(s fiber.Storage) fiber.Storage {
	if s == nil {
		return nil
	}
	return Namespace{s}
}
