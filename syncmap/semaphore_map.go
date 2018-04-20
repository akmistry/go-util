package syncmap

import (
	"context"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

type Semaphore struct {
	// According to atomic, MUST be the first element of a struct.
	refCount int64
	key      string
	m        *SemaphoreMap
	sema     *semaphore.Weighted
}

func (s *Semaphore) Release(n int64) {
	s.sema.Release(n)
	refs := atomic.AddInt64(&s.refCount, -n)

	if refs == 0 {
		s.m.tryDelete(s)
	}
}

// SemaphoreMap is a map of semaphores that automatically allocates and
// deallocates semaphores as necessary. It allows for fine-grained semephore
// use, such as resource allocations and concurrent resource limits, for a
// potentially large key space, while only consuming space propotional to the
// number of acquired semaphores.
type SemaphoreMap struct {
	n     int64
	semas map[string]*Semaphore
	lock  sync.Mutex
}

func NewSemaphoreMap(n int64) *SemaphoreMap {
	return &SemaphoreMap{n: n, semas: make(map[string]*Semaphore)}
}

func (m *SemaphoreMap) fetchSemaEmtry(key string, n int64) *Semaphore {
	m.lock.Lock()
	defer m.lock.Unlock()

	s := m.semas[key]
	if s == nil {
		s = &Semaphore{key: key, m: m, sema: semaphore.NewWeighted(m.n)}
		m.semas[key] = s
	}
	atomic.AddInt64(&s.refCount, n)
	return s
}

func (m *SemaphoreMap) Acquire(key string, n int64) *Semaphore {
	s := m.fetchSemaEmtry(key, n)

	err := s.sema.Acquire(context.Background(), n)
	if err != nil {
		panic(err)
	}
	return s
}

func (m *SemaphoreMap) AcquireContext(ctx context.Context, key string, n int64) (*Semaphore, error) {
	s := m.fetchSemaEmtry(key, n)

	err := s.sema.Acquire(ctx, n)
	if err != nil {
		refs := atomic.AddInt64(&s.refCount, -n)
		if refs == 0 {
			m.tryDelete(s)
		}
		return nil, err
	}
	return s, nil
}

func (m *SemaphoreMap) TryAcquire(key string, n int64) *Semaphore {
	s := m.fetchSemaEmtry(key, n)

	didAcquire := s.sema.TryAcquire(n)
	if !didAcquire {
		refs := atomic.AddInt64(&s.refCount, -n)
		if refs == 0 {
			m.tryDelete(s)
		}
		return nil
	}
	return s
}

func (m *SemaphoreMap) tryDelete(s *Semaphore) {
	m.lock.Lock()
	defer m.lock.Unlock()
	refs := atomic.LoadInt64(&s.refCount)
	if refs < 0 {
		panic("refCount < 0")
	} else if refs != 0 {
		return
	}

	delete(m.semas, s.key)
}
