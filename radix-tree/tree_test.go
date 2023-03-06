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
	testAscendDescend(t, &tree, 0xFF0F)
	testAscendDescend(t, &tree, 0xFF11)
}

func testAscendDescend(t *testing.T, tree *Tree, start uint64) {
	t.Helper()
	count := 0

	expectedCount := tree.Len()
	hasKey := false
	if tree.Get(Key(start)) != nil {
		hasKey = true
		expectedCount++
	}

	var prev Item
	first := true
	tree.AscendGreaterOrEqual(Key(start), func(item Item) bool {
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
	tree.DescendLessOrEqual(Key(start), func(item Item) bool {
		if hasKey && first {
			if item.Key() != start {
				t.Errorf("First key %016x != start Key %016x", item.Key(), start)
			}
		}
		if item.Key() > start {
			t.Errorf("Key %016x greater then %016x", item.Key(), start)
		} else if prev != nil && item.Key() > prev.Key() {
			t.Errorf("Key %016x greater then previous %016x", item.Key(), prev.Key())
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
		kv := tree.Get(Key(k))
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
			old := tree.Delete(Key(k))
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
		kv := tree.Get(Key(k))
		if kv == nil {
			t.Errorf("key %016x not found", k)
		} else if r, ok := kv.(*TestItem); !ok || r.val != v {
			t.Errorf("ok %v, r %016x", ok, r)
		}
		testAscendDescend(t, &tree, k)
	}
	for k := range removed {
		kv := tree.Get(Key(k))
		if kv != nil {
			t.Errorf("key %016x found", k)
		}
		testAscendDescend(t, &tree, k)
	}

	for i := 0; i < 1000; i++ {
		testAscendDescend(t, &tree, rand.Uint64())
	}
}

func generateItems(num int) []TestItem {
	items := make([]TestItem, num)
	for i := range items {
		items[i].key = rand.Uint64()
	}
	return items
}

func BenchmarkInsert(b *testing.B) {
	insertItems := 1
	for i := 0; i < 7; i++ {
		items := generateItems(insertItems)
		testName := fmt.Sprintf("%d", insertItems)
		b.Run(testName, func(b *testing.B) {
			b.ReportAllocs()
			var tree Tree
			for i := 0; i < b.N; i++ {
				if i%insertItems == 0 {
					tree.Clear()
				}
				tree.ReplaceOrInsert(&items[i%insertItems])
			}
		})
		insertItems *= 10
	}
}

func BenchmarkInsertFull(b *testing.B) {
	insertItems := 1
	for i := 0; i < 7; i++ {
		items := generateItems(insertItems)
		testName := fmt.Sprintf("%d", insertItems)
		var tree Tree
		for i := 0; i < insertItems; i++ {
			tree.ReplaceOrInsert(&items[i])
		}
		b.Run(testName, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tree.ReplaceOrInsert(&items[i%insertItems])
			}
		})
		insertItems *= 10
	}
}

func BenchmarkDeleteInsert(b *testing.B) {
	insertItems := 1
	for i := 0; i < 7; i++ {
		items := generateItems(insertItems)
		testName := fmt.Sprintf("%d", insertItems)
		var tree Tree
		for i := 0; i < insertItems; i++ {
			tree.ReplaceOrInsert(&items[i])
		}
		b.Run(testName, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				item := &items[i%insertItems]
				tree.Delete(item)
				item.key = rand.Uint64()
				tree.ReplaceOrInsert(item)
			}
		})
		insertItems *= 10
	}
}

func BenchmarkGet(b *testing.B) {
	insertItems := 1
	for i := 0; i < 7; i++ {
		items := generateItems(insertItems)
		testName := fmt.Sprintf("%d", insertItems)
		var tree Tree
		for i := 0; i < insertItems; i++ {
			tree.ReplaceOrInsert(&items[i])
		}
		b.Run(testName, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tree.Get(&items[i%insertItems])
			}
		})
		insertItems *= 10
	}
}

func BenchmarkMax(b *testing.B) {
	insertItems := 1
	for i := 0; i < 7; i++ {
		items := generateItems(insertItems)
		testName := fmt.Sprintf("%d", insertItems)
		var tree Tree
		for i := 0; i < insertItems; i++ {
			tree.ReplaceOrInsert(&items[i])
		}
		b.Run(testName, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tree.Max()
			}
		})
		insertItems *= 10
	}
}

func BenchmarkDescend(b *testing.B) {
	const DescendItems = 2

	insertItems := 1
	for i := 0; i < 7; i++ {
		items := generateItems(insertItems)
		testName := fmt.Sprintf("%d", insertItems)
		var tree Tree
		for i := 0; i < insertItems; i++ {
			tree.ReplaceOrInsert(&items[i])
		}
		b.Run(testName, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				count := 0
				tree.DescendLessOrEqual(&items[i%insertItems], func(item Item) bool {
					count++
					return count < DescendItems
				})
			}
		})
		insertItems *= 10
	}
}
