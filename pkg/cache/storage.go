package cache

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Key string
type Namespace string

func (n Namespace) Key(key string) Key {
	return Key(fmt.Sprintf("%s:::%s", n, key))
}

func (k Key) Namespace() Namespace {
	split := strings.SplitN(string(k), ":::", 2)
	if len(split) == 2 {
		return Namespace(split[0])
	}
	return ""
}

type Storage interface {
	Get(ctx context.Context, key Key) ([]byte, bool)
	Add(ctx context.Context, key Key, value []byte)
	NamespaceTTL(namepace Namespace, ttl time.Duration)
}
