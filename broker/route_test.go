package broker

import (
	"github.com/funkygao/assert"
	"testing"
)

func TestIsGlobalTopic(t *testing.T) {
	assert.Equal(t, false, isGlobalTopic("topic"))
	assert.Equal(t, false, isGlobalTopic("ra/"))
	assert.Equal(t, false, isGlobalTopic("r"))
	assert.Equal(t, true, isGlobalTopic("r/"))
	assert.Equal(t, true, isGlobalTopic("r/user/3434"))
	assert.Equal(t, false, isGlobalTopic("user/232322"))
	assert.Equal(t, true, isGlobalTopic("r/alliance/8989"))
}
