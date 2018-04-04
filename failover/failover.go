package failover

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"time"
)

const (
	DefaultTimeout = time.Second
)

var (
	// ErrFailover is the error to return from a Func to indicate failure to
	// obtain a result, and that the next available element should be tried.
	ErrFailover = errors.New("failover: fail over to next option. not an error")

	// ErrNoResult indicates that no result was available after performing all
	// available tries.
	ErrNoResult = errors.New("failover: no result available")
)

type Func = func(context.Context, interface{}) (interface{}, error)

type Options struct {
	// Time before first failover attempt. If zero, DefaultTimeout is
	// used.
	InitialTimeout time.Duration

	// Time between failover attempts after the first. If zero, InitialTimeout
	// is used. To force all failovers to occur in parallel (no staggering), use
	// a trivially small duration, such as time.Nanosecond.
	StaggerInterval time.Duration
}

type result struct {
	v   interface{}
	err error
}

func DoSlice(ctx context.Context, slice interface{}, f Func, opts *Options) (interface{}, error) {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		panic("not a slice")
	}

	ch := make(chan interface{})
	done := make(chan struct{})
	defer func() {
		<-done
	}()

	ctx, cf := context.WithCancel(ctx)
	defer cf()
	go func() {
		defer close(done)
		defer close(ch)
		for i := 0; i < rv.Len(); i++ {
			v := rv.Index(i).Interface()
			select {
			case <-ctx.Done():
				return
			case ch <- v:
			}
		}
	}()
	return Do(ctx, ch, f, opts)
}

func Do(ctx context.Context, ch <-chan interface{}, f Func, opts *Options) (interface{}, error) {
	// Intentionally unbuffered.
	resultCh := make(chan result)
	done := make(chan struct{})
	defer func() {
		// Wait for this channel to be closed to ensure all child goroutines are
		// closed.
		<-done
	}()

	ctx, cf := context.WithCancel(ctx)
	defer cf()

	go func() {
		defer close(done)
		defer close(resultCh)

		var wg sync.WaitGroup
		defer wg.Wait()
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			case arg, ok := <-ch:
				if !ok {
					// No more options.
					return
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					out, err := f(ctx, arg)
					// Ignore context errors.
					if err != nil && err == ctx.Err() {
						if cl, ok := out.(io.Closer); ok {
							cl.Close()
						}
						return
					}

					// First, try to deliver the result ignoring the context. This has an
					// interesting use case where the user can hold back a result using
					// the context, and use it as a "last resort" value.
					select {
					case resultCh <- result{out, err}:
						return
					default:
					}

					select {
					case resultCh <- result{out, err}:
					case <-ctx.Done():
						if cl, ok := out.(io.Closer); ok {
							cl.Close()
						}
					}
				}()
			}

			timeout := DefaultTimeout
			if opts != nil {
				if opts.InitialTimeout > 0 {
					timeout = opts.InitialTimeout
				}
				if i > 0 && opts.StaggerInterval > 0 {
					timeout = opts.StaggerInterval
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(timeout):
			}
		}
	}()

	for r := range resultCh {
		if r.err != ErrFailover {
			return r.v, r.err
		}
		// r.err == ErrFailover, wait for the next result.
	}

	// Channel closed, all requests completed.
	if ctx.Err() != nil {
		// We could be here because of a context done, in which case, return
		// the context error.
		return nil, ctx.Err()
	}
	return nil, ErrNoResult
}
