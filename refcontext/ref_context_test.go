package refcontext

import (
	"context"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func expectEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("result %+v != expected %+v", actual, expected)
	}
}

func TestRefContext(t *testing.T) {
	const numContexts = 1000
	ctxs := make(map[context.Context]context.CancelFunc)
	for i := 0; i < numContexts; i++ {
		ctx, cf := context.WithCancel(context.Background())
		ctxs[ctx] = cf
	}

	var crc *RefContext
	for ctx, _ := range ctxs {
		if crc == nil {
			crc, _ = New(ctx)
		} else {
			expectEqual(t, true, crc.Ref(ctx))
		}
	}

	expectEqual(t, nil, crc.Err())
	for len(ctxs) > 0 {
		for ctx, cf := range ctxs {
			// Delete a random 50% each time. This is to randomise the cancel order.
			if rand.Float64() < 0.1 {
				cf()
				delete(ctxs, ctx)
			}
		}
	}
	<-crc.Done()

	expectEqual(t, false, crc.Ref(context.Background()))
}

func TestRefContextCancel(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	defer cf()

	crc, crcCf := New(ctx)

	// Wait for the waiting goroutine to start.
	time.Sleep(time.Millisecond)
	crcCf()

	expectEqual(t, false, crc.Ref(context.Background()))
}

func TestRefContextCancelRace(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	crc, _ := New(ctx)

	go func() {
		// Make both sleep so that we have a 50/50 chance of either happening
		// first.
		time.Sleep(time.Millisecond)
		cf()
	}()
	time.Sleep(time.Millisecond)
	if !crc.Ref(context.Background()) {
		select {
		case <-ctx.Done():
		default:
			t.Error("Context should be done")
		}
	}
}
