package broker

import (
	"github.com/funkygao/assert"
	"strings"
	"testing"
)

func TestWild(t *testing.T) {
	w := newWild("ab/c/x", nil)
	assert.Equal(t, true, w.valid())

	w = newWild("abc/finance+", nil)
	assert.Equal(t, false, w.valid())

	w = newWild("abc/#", nil)
	parts := strings.Split("abc/d/e/f/g", SLASH)
	assert.Equal(t, true, w.matches(parts))
	parts = strings.Split("m/q/w/e", SLASH)
	assert.Equal(t, false, w.matches(parts))

	w = newWild("abc/+/c", nil)
	parts = strings.Split("abc/d/c", SLASH)
	assert.Equal(t, true, w.matches(parts))
	parts = strings.Split("abc/xxxx/c", SLASH)
	assert.Equal(t, true, w.matches(parts))
	parts = strings.Split("abc/d/e/c", SLASH)
	assert.Equal(t, false, w.matches(parts))
}

func TestIsWildcard(t *testing.T) {
	assert.Equal(t, true, isWildcard("a/b/+/c"))
	assert.Equal(t, true, isWildcard("SYS/broker/conns/#"))
}
