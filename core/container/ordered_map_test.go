package container_test

import (
	"github.com/stretchr/testify/assert"
	"snow/core/container"
	"strings"
	"testing"
)

func TestOrderedMap(t *testing.T) {
	m := container.NewOrderedMap(
		container.Pair[int, string]{1, "ab"},
		container.Pair[int, string]{2, "ac"},
		container.Pair[int, string]{3, "ad"},
		container.Pair[int, string]{4, "af"},
	)

	assert.Condition(
		t,
		func() (success bool) {
			success = true
			m.ScanIf(func(entry container.Pair[int, string]) bool {
				if !strings.ContainsAny(entry.Second, "a") {
					success = false
					return false
				}

				return true
			})
			return
		},
		"ScanIf can scan all elements that satisfy predicts",
	)

	newMap := container.NewOrderedMap[int, string]()
	m.Scan(func(entry container.Pair[int, string]) {
		newMap.Add(entry.First, entry.Second)
	})
	assert.EqualValues(t, m, newMap, "ScanIf can scan all elements that satisfy predicts")

	assert.Condition(
		t,
		func() (success bool) {
			success = true
			m.ScanKVIf(func(_ int, v string) bool {
				if !strings.ContainsAny(v, "a") {
					success = false
					return false
				}

				return true
			})
			return
		},
		"ScanKVIf can scan all elements that satisfy predicts",
	)

	newMap = container.NewOrderedMap[int, string]()
	m.ScanKV(func(first int, value string) {
		newMap.Add(first, value)
	})
	assert.EqualValues(t, m, newMap, "ScanKV can scan all elements that satisfy predicts")

	assert.True(t, m.Len() == 4, "The length of the ordered map must match the initializer arguments' length specified")

	newMap = container.NewOrderedMap[int, string]()
	newMap.AddAll(m)
	assert.EqualValues(t, m, newMap, "Give an empty ordered map and add value as specified")

	resultMap := container.NewOrderedMap(
		container.Pair[int, string]{2, "ac"},
		container.Pair[int, string]{3, "ad"},
		container.Pair[int, string]{4, "af"},
	)
	newMap = container.NewOrderedMap[int, string]()
	newMap.AddAll(m)
	newMap.Remove(1)
	assert.EqualValues(t, resultMap, newMap, "Ordered map remove give key then map must match specified")

	newMap = container.NewOrderedMap[int, string]()
	newMap.AddAll(m)
	keys := container.NewList(1, 2, 3, 4)
	newMap.RemoveAll(keys)
	assert.Empty(t, newMap, "Map Remove give keys then map must match specified")

	keys = m.Keys()
	assert.ElementsMatch(t, keys, container.List[int]{1, 2, 3, 4}, "Get ordered map all keys,keys must match specified")

	values := m.Values()
	assert.ElementsMatch(t, values, container.List[string]{"ab", "ac", "ad", "af"}, "Get ordered map all keys,keys must match specified")

	found := -1
	assert.Condition(
		t,
		func() (success bool) {
			found = m.WithKey(1, func(key int, val string) {
				success = key == 1 && val == "ab"
			})
			return
		},
		"Give one map and make use of WithKey() member",
	)
	assert.True(t, found == 1, "use of WithKey() member expect 1 ")

	assert.Condition(
		t,
		func() (success bool) {
			foundKeyList := []int{}
			foundValList := []string{}
			found = m.WithKeyRange(2, 4, func(key int, val string) {
				foundKeyList = append(foundKeyList, key)
				foundValList = append(foundValList, val)
			})
			assert.ElementsMatch(t, foundValList, container.List[string]{"ac", "ad", "af"}, "values must match specified")
			assert.ElementsMatch(t, foundKeyList, []int{2, 3, 4}, "keys must match specified")

			return true
		},
		"Give one map and make use of WithKeyRange() member",
	)

	keys = []int{}
	m.Scan(func(entry container.Pair[int, string]) {
		keys.Add(entry.First)
	})

	assert.Equal(t, container.List[int]{1, 2, 3, 4}, keys, "order map is sequence  match specified")
}
