package syncmap

import (
	"sync"
	"sync/atomic"
)

type Lock struct {
	key      string
	m        *LockMap
	refCount int32
	lock     sync.Mutex
}

func (l *Lock) Unlock() {
	l.lock.Unlock()
	refs := atomic.AddInt32(&l.refCount, -1)

	if refs == 0 {
		l.m.tryDelete(l)
	}
}

// LockMap is a map of mutexes that automatically allocates and deallocates
// mutexes as necessary. It allows for fine-grained mutual exclusion for a
// potentially large key space, while only consuming space propotional to the
// number of acquired mutexes. The zero-initialised LockMap is ready to use.
type LockMap struct {
	locks map[string]*Lock
	lock  sync.Mutex
}

func (m *LockMap) lazyInit() {
	if m.locks == nil {
		m.locks = make(map[string]*Lock)
	}
}

func (m *LockMap) Lock(key string) *Lock {
	m.lock.Lock()
	m.lazyInit()
	l := m.locks[key]
	if l == nil {
		l = &Lock{key: key, m: m}
		m.locks[key] = l
	}
	atomic.AddInt32(&l.refCount, 1)
	m.lock.Unlock()

	l.lock.Lock()
	return l
}

func (m *LockMap) tryDelete(l *Lock) {
	m.lock.Lock()
	defer m.lock.Unlock()
	refs := atomic.LoadInt32(&l.refCount)
	if refs < 0 {
		panic("refCount < 0")
	} else if refs != 0 {
		return
	}

	delete(m.locks, l.key)
}
