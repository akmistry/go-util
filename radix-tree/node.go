package radix

import (
	"github.com/akmistry/go-util/bitmap"
)

type node struct {
	// Prefix. Prefix length is stored in the low 8 bits.
	prefix uint64

	itemBitmap bitmap.Bitmap256
	cItemList  []Item
}

func prefixMask(prefixLen int) uint64 {
	return ^((uint64(1) << ((8 - prefixLen) * 8)) - 1)
}

func commonPrefixLen(a, b uint64) int {
	const topMask = 0xFF00000000000000
	c := a ^ b
	zeros := 0
	for i := 0; i < 8; i++ {
		if c&topMask != 0 {
			break
		}
		zeros++
		c <<= 8
	}
	return zeros
}

func makeNewNode(prefix uint64, prefixLen int) *node {
	prefix &= prefixMask(prefixLen)
	return &node{
		prefix: prefix | uint64(prefixLen&0xFF),
		// New nodes will always have 2 elements added to it.
		cItemList: make([]Item, 0, 2),
	}
}

func (n *node) Key() uint64 {
	return n.prefix & n.prefixMask()
}

func (n *node) prefixLen() int {
	return int(n.prefix & 0xFF)
}

func (n *node) prefixMask() uint64 {
	return prefixMask(int(n.prefix & 0xFF))
}

func (n *node) matchesPrefix(key uint64) bool {
	return (key^n.prefix)&n.prefixMask() == 0
}

func (n *node) empty() bool {
	return len(n.cItemList) == 0
}

func (n *node) index(key uint64) uint8 {
	return uint8((key >> (8 * (7 - n.prefixLen()))) & 0xFF)
}

func (n *node) getChild(index uint8) (Item, int) {
	i := n.itemBitmap.CountLess(index)
	if i < len(n.cItemList) && n.itemBitmap.Get(index) {
		return n.cItemList[i], i
	}
	return nil, i
}

func (n *node) insertChildInto(index uint8, item Item, slot int) {
	i := slot
	if i < len(n.cItemList) && n.itemBitmap.Get(index) {
		if item == nil {
			n.itemBitmap.Clear(index)

			copy(n.cItemList[i:], n.cItemList[i+1:])
			// Nil out the last element so that the GC can free it
			//n.cItemList[len(n.cItemList)-1] = nil
			n.cItemList = n.cItemList[:len(n.cItemList)-1]

			if len(n.cItemList) > 2 && len(n.cItemList) < (cap(n.cItemList)/3) {
				n.cItemList = append([]Item(nil), n.cItemList...)
			}
		} else {
			n.cItemList[i] = item
		}
	} else if item != nil {
		n.itemBitmap.Set(index)

		if i < len(n.cItemList) {
			n.cItemList = append(n.cItemList, nil)
			copy(n.cItemList[i+1:], n.cItemList[i:])
			n.cItemList[i] = item
		} else {
			n.cItemList = append(n.cItemList, item)
		}
	}
}

func (n *node) fetch(key uint64) Item {
	for {
		if !n.matchesPrefix(key) {
			// Lookup prefix does not match this node's, therefore key not found.
			return nil
		}

		i := n.index(key)
		next, _ := n.getChild(i)
		if next == nil {
			return nil
		}
		if nn, ok := next.(*node); ok {
			n = nn
			continue
		}
		if next.Key() != key {
			return nil
		}
		return next
	}
}

func makeNodeFromItems(a, b Item) *node {
	aKey := a.Key()
	bKey := b.Key()
	commonPrefix := commonPrefixLen(aKey, bKey)

	newNode := makeNewNode(aKey, commonPrefix)
	if aKey < bKey {
		newNode.insertChildInto(newNode.index(aKey), a, 0)
		newNode.insertChildInto(newNode.index(bKey), b, 1)
	} else {
		newNode.insertChildInto(newNode.index(bKey), b, 0)
		newNode.insertChildInto(newNode.index(aKey), a, 1)
	}

	return newNode
}

func (n *node) insert(item Item) Item {
	key := item.Key()
	for {
		if !n.matchesPrefix(key) {
			panic("Key does not belong on this node")
		}

		index := n.index(key)
		next, slot := n.getChild(index)
		if next == nil {
			n.insertChildInto(index, item, slot)
			return nil
		}
		nn, ok := next.(*node)
		if ok && nn.matchesPrefix(key) {
			n = nn
			continue
		} else if !ok && next.Key() == key {
			n.insertChildInto(index, item, slot)
			return next
		}

		// Create a new node with a common prefix
		newNode := makeNodeFromItems(item, next)
		n.insertChildInto(index, newNode, slot)
		return nil
	}
}

func (n *node) delete(key uint64) Item {
	if !n.matchesPrefix(key) {
		// Item not here
		return nil
	}

	index := n.index(key)
	next, slot := n.getChild(index)
	if next == nil {
		// Item not found, nothing to delete
		return nil
	}
	if nn, ok := next.(*node); ok {
		old := nn.delete(key)
		if nn.empty() {
			n.insertChildInto(index, nil, slot)
		}
		return old
	}
	if next.Key() == key {
		n.insertChildInto(index, nil, slot)
		return next
	}

	// Item not found, nothing to delete
	return nil
}

func (n *node) max() Item {
	for {
		if n.empty() {
			return nil
		}

		last := n.cItemList[len(n.cItemList)-1]
		if nn, ok := last.(*node); ok {
			n = nn
			continue
		}
		return last
	}
}

func (n *node) min() Item {
	for {
		if n.empty() {
			return nil
		}

		first := n.cItemList[0]
		if nn, ok := first.(*node); ok {
			n = nn
			continue
		}
		return first
	}
}

func (n *node) ascendGreaterOrEqual(key uint64, iter IterFunc) bool {
	i := 0
	if n.matchesPrefix(key) {
		// Look for the first index to start iterating
		i = n.itemBitmap.CountLess(n.index(key))
	}

	for ; i < len(n.cItemList); i++ {
		child := n.cItemList[i]
		if nn, ok := child.(*node); ok {
			if !nn.ascendGreaterOrEqual(key, iter) {
				return false
			}
		} else if child.Key() >= key {
			if !iter(child) {
				return false
			}
		}
	}
	return true
}

func (n *node) descendLessOrEqual(key uint64, iter IterFunc) bool {
	i := len(n.cItemList) - 1
	if n.matchesPrefix(key) {
		// Look for the first index to start iterating
		firstIndex := n.index(key)
		i = n.itemBitmap.CountLess(firstIndex) - 1
		if n.itemBitmap.Get(firstIndex) {
			i++
		}
	}

	for ; i >= 0; i-- {
		child := n.cItemList[i]
		if nn, ok := child.(*node); ok {
			if !nn.descendLessOrEqual(key, iter) {
				return false
			}
		} else if child.Key() <= key {
			if !iter(child) {
				return false
			}
		}
	}
	return true
}
