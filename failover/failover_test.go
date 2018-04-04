package failover

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

var opts = &Options{
	InitialTimeout: 13 * time.Millisecond,
}

func expectEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("result %+v != expected %+v", actual, expected)
	}
}

func TestFailover(t *testing.T) {
	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		i := iface.(int)
		if i == 1 {
			return 1, nil
		}
		return nil, ErrFailover
	}

	ch := make(chan interface{}, 3)
	ch <- 0
	ch <- 1
	ch <- 2
	close(ch)

	startTime := time.Now()
	r, err := Do(context.Background(), ch, f, opts)
	expectEqual(t, nil, err)
	expectEqual(t, 1, r)
	if time.Since(startTime) < opts.InitialTimeout {
		t.Errorf("failover time %v too short", time.Since(startTime))
	}
}

func TestFailoverSlice(t *testing.T) {
	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		i := iface.(int)
		if i == 1 {
			return 1, nil
		}
		return nil, ErrFailover
	}

	startTime := time.Now()
	r, err := DoSlice(context.Background(), []int{0, 1, 2}, f, opts)
	expectEqual(t, nil, err)
	expectEqual(t, 1, r)
	if time.Since(startTime) < opts.InitialTimeout {
		t.Errorf("failover time %v too short", time.Since(startTime))
	}
}

func TestFailoverSliceEmpty(t *testing.T) {
	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		panic("unexpected call")
		return nil, ErrFailover
	}

	r, err := DoSlice(context.Background(), []int{}, f, opts)
	expectEqual(t, ErrNoResult, err)
	expectEqual(t, nil, r)
}

func TestFailoverFirstResult(t *testing.T) {
	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		i := iface.(int)
		if i == 0 {
			return 0, nil
		}
		return nil, ErrFailover
	}

	ch := make(chan interface{}, 2)
	ch <- 0
	ch <- 1
	close(ch)

	startTime := time.Now()
	r, err := Do(context.Background(), ch, f, opts)
	expectEqual(t, nil, err)
	expectEqual(t, 0, r)
	if time.Since(startTime) >= opts.InitialTimeout {
		t.Errorf("failover time %v too long", time.Since(startTime))
	}
}

func TestFailoverNoResult(t *testing.T) {
	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		return nil, ErrFailover
	}

	ch := make(chan interface{}, 2)
	ch <- 0
	ch <- 1
	close(ch)

	r, err := Do(context.Background(), ch, f, opts)
	expectEqual(t, ErrNoResult, err)
	expectEqual(t, nil, r)
}

func TestFailoverError(t *testing.T) {
	errTest := errors.New("test error")
	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		if iface.(int) == 0 {
			return nil, errTest
		}
		return nil, ErrFailover
	}

	ch := make(chan interface{}, 2)
	ch <- 0
	ch <- 1
	close(ch)

	startTime := time.Now()
	r, err := Do(context.Background(), ch, f, opts)
	expectEqual(t, errTest, err)
	expectEqual(t, nil, r)
	if time.Since(startTime) >= opts.InitialTimeout {
		t.Errorf("failover too long: %v", time.Since(startTime))
	}
}

func TestFailoverTimeout(t *testing.T) {
	testTimeout := opts.InitialTimeout * 3
	ctx, cf := context.WithTimeout(context.Background(), testTimeout)
	defer cf()

	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	startTime := time.Now()
	r, err := DoSlice(ctx, []int{0, 1}, f, opts)
	expectEqual(t, context.DeadlineExceeded, err)
	expectEqual(t, nil, r)
	if time.Since(startTime) < testTimeout {
		t.Errorf("failover too short: %v", time.Since(startTime))
	}
}

func TestFailoverTimeoutFailoverWait(t *testing.T) {
	testTimeout := opts.InitialTimeout * 3
	ctx, cf := context.WithTimeout(context.Background(), testTimeout)
	defer cf()

	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	ch := make(chan interface{}, 1)
	ch <- 0
	defer close(ch)

	startTime := time.Now()
	r, err := Do(ctx, ch, f, opts)
	expectEqual(t, context.DeadlineExceeded, err)
	expectEqual(t, nil, r)
	if time.Since(startTime) < testTimeout {
		t.Errorf("failover too short: %v", time.Since(startTime))
	}
}

type closedType bool

func (t *closedType) Close() error {
	*t = true
	return nil
}

func TestFailoverTimeoutClosed(t *testing.T) {
	testTimeout := opts.InitialTimeout * 3
	ctx, cf := context.WithTimeout(context.Background(), testTimeout)
	defer cf()

	c0 := new(closedType)
	c1 := new(closedType)
	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		<-ctx.Done()
		return iface, nil
	}

	startTime := time.Now()
	r, err := DoSlice(ctx, [](*closedType){c0, c1}, f, opts)
	expectEqual(t, nil, err)
	if r == nil {
		t.Error("result nil")
	} else if *r.(*closedType) {
		t.Error("value unexpectedly closed")
	}
	if *c0 == *c1 {
		t.Errorf("c0 %v == c1 %v", *c0, *c1)
	}
	if time.Since(startTime) < testTimeout {
		t.Errorf("failover too short: %v", time.Since(startTime))
	}
}

func TestFailoverTimeoutLastResort(t *testing.T) {
	testTimeout := opts.InitialTimeout * 3
	ctx, cf := context.WithTimeout(context.Background(), testTimeout)
	defer cf()

	closed := new(closedType)
	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		<-ctx.Done()
		return closed, nil
	}

	startTime := time.Now()
	r, err := DoSlice(ctx, []int{0}, f, opts)
	expectEqual(t, nil, err)
	expectEqual(t, closed, r)
	if *r.(*closedType) {
		t.Error("value unexpectedly closed")
	}
	if time.Since(startTime) < testTimeout {
		t.Errorf("failover too short: %v", time.Since(startTime))
	}
}

func TestFailoverStagger(t *testing.T) {
	testOpts := *opts
	testOpts.StaggerInterval = 5 * time.Millisecond

	startTime := time.Now()
	f := func(ctx context.Context, iface interface{}) (interface{}, error) {
		i := iface.(int)
		expectedDelay := time.Duration(i-1)*testOpts.StaggerInterval + testOpts.InitialTimeout
		maxDelay := time.Duration(i)*testOpts.StaggerInterval + testOpts.InitialTimeout
		if i > 0 && time.Since(startTime) < expectedDelay {
			t.Errorf("failover delay %v < expected %v", time.Since(startTime), expectedDelay)
		} else if i > 0 && time.Since(startTime) > maxDelay {
			t.Errorf("failover delay %v too long, expected %v", time.Since(startTime), expectedDelay)
		}
		return nil, ErrFailover
	}

	r, err := DoSlice(context.Background(), []int{0, 1, 2}, f, &testOpts)
	expectEqual(t, ErrNoResult, err)
	expectEqual(t, nil, r)
}
