package refcontext

import (
	"context"
	"math/rand"
	"sync"
)

type RefContext struct {
	context.Context
	cancel context.CancelFunc

	ctxs []context.Context
	lock sync.Mutex
}

func New(initialCtx context.Context) (*RefContext, context.CancelFunc) {
	ctx, cf := context.WithCancel(context.Background())
	c := &RefContext{
		Context: ctx,
		cancel:  cf,
		ctxs:    []context.Context{initialCtx},
	}
	go c.watchCtx()
	return c, cf
}

func (c *RefContext) watchCtx() {
	c.lock.Lock()
	defer c.lock.Unlock()

	defer func() {
		c.ctxs = nil
	}()

	for {
		ctx := c.ctxs[0]

		c.lock.Unlock()
		select {
		case <-ctx.Done():
		case <-c.Done():
		}
		c.lock.Lock()

		if c.Err() != nil {
			// Must do this here because of the deferred Mutex.Unlock().
			return
		}

		if c.ctxs[0] != ctx {
			panic("ctx != ctxs[0]")
		}

		last := len(c.ctxs) - 1
		c.ctxs[0] = c.ctxs[last]
		c.ctxs[last] = nil
		c.ctxs = c.ctxs[:last]

		// Cleanup before checking capacity, to maximise ability to free up space.
		c.randCleanup()
		if len(c.ctxs) == 0 {
			c.cancel()
			return
		} else if cap(c.ctxs) > len(c.ctxs)*2 {
			// Cleanup free space if capacity is too big.
			c.ctxs = append([]context.Context{}, c.ctxs...)
		}
	}
}

func (c *RefContext) randCleanup() {
	for len(c.ctxs) > 1 {
		// Ignore c.ctxs[0] becuase it might currently be waited on by the waiter
		// goroutine.
		i := rand.Intn(len(c.ctxs)-1) + 1

		select {
		case <-c.ctxs[i].Done():
			// Swap with last and delete.
		default:
			return
		}

		last := len(c.ctxs) - 1
		c.ctxs[i] = c.ctxs[last]
		c.ctxs[last] = nil
		c.ctxs = c.ctxs[:last]
	}
}

func (c *RefContext) Ref(ctx context.Context) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if len(c.ctxs) == 0 || c.Err() != nil {
		return false
	}

	c.ctxs = append(c.ctxs, ctx)
	c.randCleanup()
	return true
}
