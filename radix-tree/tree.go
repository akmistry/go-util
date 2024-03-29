package radix

import (
	"math"
)

// Tree implements a radix tree (https://en.wikipedia.org/wiki/Radix_tree)
// with uint64 keys. This implementation is intended to be cache-efficient,
// minimising the number of cache line accesses (and hence misses).
//
// The API follows google.BTree, with the exception that the Item interface
// provides a uint64 key instead of a Less() function.
//
// The zero-value Tree is ready to use.
//
// NOTE: Only a subset of functions have been implemented.
type Tree struct {
	root node
	len  int
}

func (t *Tree) Ascend(iter IterFunc) {
	t.root.ascendGreaterOrEqual(0, iter)
}

func (t *Tree) AscendGreaterOrEqual(item Item, iter IterFunc) {
	t.root.ascendGreaterOrEqual(item.Key(), iter)
}

func (t *Tree) AscendGreaterOrEqualI(key uint64, iter IterFunc) {
	t.root.ascendGreaterOrEqual(key, iter)
}

func (t *Tree) Clear() {
	t.root = node{}
	t.len = 0
}

func (t *Tree) Descend(iter IterFunc) {
	t.root.descendLessOrEqual(math.MaxUint64, iter)
}

func (t *Tree) DescendLessOrEqual(item Item, iter IterFunc) {
	t.root.descendLessOrEqual(item.Key(), iter)
}

func (t *Tree) DescendLessOrEqualI(key uint64, iter IterFunc) {
	t.root.descendLessOrEqual(key, iter)
}

func (t *Tree) Delete(key Item) Item {
	return t.DeleteI(key.Key())
}

func (t *Tree) DeleteI(key uint64) Item {
	old := t.root.delete(key)
	if old != nil {
		t.len--
	}
	return old
}

func (t *Tree) Get(key Item) Item {
	return t.root.fetch(key.Key())
}

func (t *Tree) GetI(key uint64) Item {
	return t.root.fetch(key)
}

func (t *Tree) Len() int {
	return t.len
}

func (t *Tree) Max() Item {
	return t.root.max()
}

func (t *Tree) Min() Item {
	return t.root.min()
}

func (t *Tree) ReplaceOrInsert(item Item) Item {
	old := t.root.insert(item)
	if old == nil {
		t.len++
	}
	return old
}

// Item represents a single object in the tree, with a uint64 key.
type Item interface {
	Key() uint64
}

type Key uint64

func (k Key) Key() uint64 {
	return uint64(k)
}

// IterFunc allows callers to iterate over the tree with Ascend/Descend*
// functions. Iteration will stop when this function returns false.
type IterFunc func(Item) bool
