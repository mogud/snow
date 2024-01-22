package container_test

import (
	"gitee.com/mogud/snow/core/container"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestMap(t *testing.T) {
	m := container.NewMap(
		container.Pair[int, string]{1, "ab"},
		container.Pair[int, string]{2, "ac"},
		container.Pair[int, string]{3, "ad"},
		container.Pair[int, string]{4, "af"},
	)

	isAllContains := true
	m.ScanIf(func(entry container.Pair[int, string]) bool {
		if !strings.Contains(entry.Second, "a") {
			isAllContains = false
			return false
		}
		return true
	})
	assert.True(t, isAllContains, "ScanIf can scan all elements that satisfy predicts")

	newMap := container.NewMap[int, string]()
	m.Scan(func(entry container.Pair[int, string]) {
		newMap.Add(entry.First, entry.Second)
	})
	assert.EqualValues(t, m, newMap, "Scan can iterates all elements")

	newMap = container.NewMap[int, string]()
	m.ScanKVIf(func(key int, value string) bool {
		if !strings.Contains(value, "a") {

			return false
		}
		newMap.Add(key, value)
		return true
	})
	assert.EqualValues(t, m, newMap, "ScanKVIf can scan all elements that satisfy predicts")

	newMap = container.NewMap[int, string]()
	m.ScanKV(func(key int, value string) {
		newMap.Add(key, value)
	})
	assert.EqualValues(t, m, newMap, "ScanKV can iterates all elements")

	assert.True(t, m.Len() == 4, "The length of the map must match the initializer arguments' length specified")

	newMap = container.NewMap[int, string]()
	m.ScanKV(func(key int, value string) {
		newMap.Add(key, value)
	})
	assert.EqualValues(t, m, newMap, "ScanKV can iterates all elements")

	assert.True(t, m.Len() == 4, "The length of the map must match the initializer arguments' length specified")

	newMap = container.NewMap[int, string]()
	newMap.AddAll(m)
	assert.EqualValues(t, m, newMap, "Give an empty map and add all value as specified")

	resultMap := container.NewMap(
		container.Pair[int, string]{2, "ac"},
		container.Pair[int, string]{3, "ad"},
		container.Pair[int, string]{4, "af"},
	)
	newMap.AddAll(m)
	newMap.Remove(1)
	assert.EqualValues(t, resultMap, newMap, "Map Remove give key then map must match specified")

	newMap = container.NewMap[int, string]()
	newMap.AddAll(m)
	keys := m.Keys()
	newMap.RemoveAll(keys)
	assert.Empty(t, newMap, "Map Remove give keys then map must match specified")

	assert.ElementsMatchf(t, keys, container.NewList(1, 2, 3, 4), "Get map all keys,keys must match specified")

	values := m.Values()
	assert.Condition(t, func() bool {
		result := true
		m.ScanKVIf(func(_ int, v string) bool {
			isContain := false
			values.ScanIf(func(e string) bool {
				if e == v {
					isContain = true
					return false
				}
				return true
			})

			if !isContain {
				result = false
				return false
			}

			return true
		})

		if !result {
			return result
		}

		values.ScanIf(func(e string) bool {
			isContain := false
			m.ScanKVIf(func(_ int, v string) bool {
				if e == v {
					isContain = true
					return false
				}
				return true
			})

			if !isContain {
				result = false
				return false
			}

			return true
		})

		return result
	}, "Get map all values, values must contain map value")

	newMap = container.NewMap(
		container.Pair[int, string]{2, "ac"},
		container.Pair[int, string]{3, "ad"},
		container.Pair[int, string]{4, "af"},
	)
	newMap.Clear()
	assert.Empty(t, newMap, "Map clear then map must match specified")

	foundInner := false
	m.WithKey(1, func(key int, val string) {
		foundInner = key == 1 && val == "ab"
	})
	assert.True(t, foundInner, "Give one map and make use of WithKey() member the result must match specified")

}
