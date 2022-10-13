package radix

import (
	"fmt"
	"math/rand"
	"testing"
)

type TestItem struct {
	key, val uint64
}

func (k *TestItem) Key() uint64 {
	return k.key
}

func (k *TestItem) String() string {
	return fmt.Sprintf("Key: %016x", k.key)
}

func TestTree(t *testing.T) {
	var tree Tree
	tree.ReplaceOrInsert(&TestItem{0, 0})
	tree.ReplaceOrInsert(&TestItem{1, 1})
	tree.ReplaceOrInsert(&TestItem{0xFF00, 0xFF00})
	tree.ReplaceOrInsert(&TestItem{0xFF10, 0xFF10})

	testAscendDescend(t, &tree, 0)
	testAscendDescend(t, &tree, 1)
	testAscendDescend(t, &tree, 2)
	testAscendDescend(t, &tree, 0xFE00)
	testAscendDescend(t, &tree, 0xFF00)
	testAscendDescend(t, &tree, 0xFF01)
	testAscendDescend(t, &tree, 0xFF11)
}

func testAscendDescend(t *testing.T, tree *Tree, start uint64) {
	t.Helper()
	count := 0

	startKey := &TestItem{start, 0}

	expectedCount := tree.Len()
	hasKey := false
	if tree.Get(startKey) != nil {
		hasKey = true
		expectedCount++
	}

	var prev Item
	first := true
	tree.AscendGreaterOrEqual(startKey, func(item Item) bool {
		if hasKey && first {
			if item.Key() != start {
				t.Errorf("First key %v != start Key %016x", item, start)
			}
		}
		if item.Key() < start {
			t.Errorf("Key %016x less then %016x", item.Key(), start)
		} else if prev != nil && item.Key() < prev.Key() {
			t.Errorf("Key %v less then previous %v", item, prev)
		}
		prev = item
		count++
		first = false
		return true
	})

	prev = nil
	first = true
	tree.DescendLessOrEqual(startKey, func(item Item) bool {
		if hasKey && first {
			if item.Key() != start {
				t.Errorf("First key %v != start Key %016x", item, start)
			}
		}
		if item.Key() > start {
			t.Errorf("Key %v greater then %v", item, start)
		} else if prev != nil && item.Key() > prev.Key() {
			t.Errorf("Key %v greater then previous %v", item, prev)
		}
		prev = item
		count++
		first = false
		return true
	})

	if count != expectedCount {
		t.Errorf("Unexpected iterations: %d, expected %d", count, expectedCount)
	}
}

func TestTreeStress(t *testing.T) {
	rand.Seed(1)
	const count = 12345

	keys := make(map[uint64]uint64)

	var tree Tree
	for i := 0; i < count; i++ {
		k := rand.Uint64()
		v := rand.Uint64()
		keys[k] = v
		tree.ReplaceOrInsert(&TestItem{k, v})
	}

	for k, v := range keys {
		key := &TestItem{k, 0}
		kv := tree.Get(key)
		if kv == nil {
			t.Errorf("key %016x not found", k)
		} else if r, ok := kv.(*TestItem); !ok || r.val != v {
			t.Errorf("ok %v, r %016x", ok, r)
		}
		testAscendDescend(t, &tree, k)
	}

	removed := make(map[uint64]bool)
	for k, v := range keys {
		// Remove ~50% of keys.
		if rand.Uint64()%2 == 0 {
			key := &TestItem{k, 0}
			old := tree.Delete(key)
			if old == nil {
				t.Errorf("key %016x not found", k)
			} else if r, ok := old.(*TestItem); !ok || r.val != v {
				t.Errorf("ok %v, r %016x", ok, r)
			}
			removed[k] = true
			delete(keys, k)
		}
	}

	for k, v := range keys {
		key := &TestItem{k, 0}
		kv := tree.Get(key)
		if kv == nil {
			t.Errorf("key %016x not found", k)
		} else if r, ok := kv.(*TestItem); !ok || r.val != v {
			t.Errorf("ok %v, r %016x", ok, r)
		}
		testAscendDescend(t, &tree, k)
	}
	for k := range removed {
		key := &TestItem{k, 0}
		kv := tree.Get(key)
		if kv != nil {
			t.Errorf("key %016x found", k)
		}
		testAscendDescend(t, &tree, k)
	}

	for i := 0; i < 1000; i++ {
		testAscendDescend(t, &tree, rand.Uint64())
	}
}