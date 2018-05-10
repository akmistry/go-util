package movingcounter

import (
	"testing"
	"time"
)

type testClock time.Time

func newTestClock(t time.Time) *testClock {
	c := new(testClock)
	*c = testClock(t)
	return c
}

func (c *testClock) Now() time.Time {
	return time.Time(*c)
}

func (c *testClock) advance(d time.Duration) {
	*c = testClock(c.Now().Add(d))
}

func TestCounter(t *testing.T) {
	clock := newTestClock(time.Now())
	c := NewMovingCounter(clock, time.Minute, 100, Int64Value(0))
	c.Add(Int64Value(5))
	clock.advance(time.Nanosecond)
	c.Add(Int64Value(13))
	clock.advance(time.Nanosecond)
	c.Add(Int64Value(7))
	clock.advance(time.Nanosecond)

	total, count := c.Total()
	if count != 3 {
		t.Errorf("count %d != 3", count)
	}
	if total.(Int64Value) != 25 {
		t.Errorf("total %d != 25", total.(Int64Value))
	}
	min := c.Min()
	if min.(Int64Value) != 5 {
		t.Errorf("min %d != 5", min.(Int64Value))
	}
	max := c.Max()
	if max.(Int64Value) != 13 {
		t.Errorf("max %d != 13", max.(Int64Value))
	}

	clock.advance(time.Second)
	c.Add(Int64Value(3))
	if total, count := c.Total(); count != 4 || total.(Int64Value) != 28 {
		t.Errorf("count %d, total %d", count, total.(Int64Value))
	}
	if min := c.Min().(Int64Value); min != 3 {
		t.Errorf("min %d != 3", min)
	}
	if max := c.Max().(Int64Value); max != 13 {
		t.Errorf("max %d != 13", max)
	}

	clock.advance(59 * time.Second)
	if total, count := c.Total(); count != 1 || total.(Int64Value) != 3 {
		t.Errorf("count %d, total %d", count, total.(Int64Value))
	}
	if min := c.Min().(Int64Value); min != 3 {
		t.Errorf("min %d != 3", min)
	}
	if max := c.Max().(Int64Value); max != 3 {
		t.Errorf("max %d != 3", max)
	}

	clock.advance(time.Second)
	if total, count := c.Total(); count != 0 || total.(Int64Value) != 0 {
		t.Errorf("count %d, total %d", count, total.(Int64Value))
	}
	if min := c.Min().(Int64Value); min != 0 {
		t.Errorf("min %d != 0", min)
	}
	if max := c.Max().(Int64Value); max != 0 {
		t.Errorf("max %d != 0", max)
	}
}
