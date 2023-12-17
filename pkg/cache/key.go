package cache

import (
	"strings"
)

type Key string
type Namespace string

func (n Namespace) Key(key ...string) Key {
	var sb strings.Builder
	sb.WriteString(string(n))
	sb.WriteString(":::")
	for i, k := range key {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(k)
	}

	return Key(sb.String())
}

func (k Key) Namespace() Namespace {
	split := strings.SplitN(string(k), ":::", 2)
	if len(split) == 2 {
		return Namespace(split[0])
	}
	return ""
}
