package cache

import (
	"context"
	"time"

	"github.com/shihray/gserver/selector"
)

const (
	Create string = "create"
	Update string = "update"
	Delete string = "delete"
)

type ttlKey struct{}

// Set the cache ttl
func TTL(t time.Duration) selector.Option {
	return func(o *selector.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, ttlKey{}, t)
	}
}
