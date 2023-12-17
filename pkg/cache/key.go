package cache

import (
	"fmt"
	"strings"
)

type Key string
type Namespace string

func (n Namespace) Key(key ...string) Key {
	return Key(fmt.Sprintf("%s:::%s", n, strings.Join(key, " ")))
}

func (k Key) Namespace() Namespace {
	split := strings.SplitN(string(k), ":::", 2)
	if len(split) == 2 {
		return Namespace(split[0])
	}
	return ""
}
