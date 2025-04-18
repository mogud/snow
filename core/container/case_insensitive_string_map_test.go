package container_test

import (
	"github.com/stretchr/testify/assert"
	"snow/core/container"
	"testing"
)

func TestCaseInsensitiveStringMap(t *testing.T) {
	m := container.NewCaseInsensitiveStringMap[string]()
	m.Add("a", "b")
	m.Add("A", "B")
	m.Add("c", "d")
	m.Add("g", "h")

	assert.True(t, m.Contains("a"))
	assert.True(t, m.Contains("A"))
	assert.True(t, m.Contains("c"))
	assert.True(t, m.Contains("C"))
	assert.False(t, m.Contains("e"))
	assert.False(t, m.Contains("E"))
	assert.True(t, m.Contains("g"))
	assert.Equal(t, "B", m.Get("a"))
	assert.Equal(t, "B", m.Get("A"))
	assert.Equal(t, "d", m.Get("c"))
	assert.Equal(t, "d", m.Get("C"))
	assert.Equal(t, "", m.Get("e"))
	assert.Equal(t, "", m.Get("E"))
	assert.Equal(t, "h", m.Get("g"))

	k, ok := m.TryGet("a")
	assert.True(t, ok)
	assert.Equal(t, "B", k)

	k, ok = m.TryGet("A")
	assert.True(t, ok)
	assert.Equal(t, "B", k)

	k, ok = m.TryGet("e")
	assert.False(t, ok)
	assert.Equal(t, "", k)

	assert.Equal(t, 3, m.Len())
	m.Remove("G")
	assert.Equal(t, 2, m.Len())
	assert.False(t, m.Contains("g"))

}
