package cache

import (
	"context"
	"time"
)

type Storage interface {
	Get(ctx context.Context, key Key) ([]byte, bool)
	Add(ctx context.Context, key Key, value []byte)
	NamespaceTTL(namepace Namespace, ttl time.Duration)
}
