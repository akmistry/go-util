package movingcounter

import (
	"time"
)

type Clock interface {
	Now() time.Time
}

type defaultClock struct{}

func (defaultClock) Now() time.Time {
	return time.Now()
}

type Value interface {
	Add(a Value) Value
	Sub(a Value) Value
	Min(a Value) Value
	Max(a Value) Value
}
type Int64Value int64

func (v Int64Value) Add(a Value) Value {
	return v + a.(Int64Value)
}

func (v Int64Value) Sub(a Value) Value {
	return v - a.(Int64Value)
}

func (v Int64Value) Min(a Value) Value {
	if a.(Int64Value) < v {
		return a
	}
	return v
}

func (v Int64Value) Max(a Value) Value {
	if a.(Int64Value) > v {
		return a
	}
	return v
}

type MovingCounter struct {
	clock                Clock
	period, bucketPeriod time.Duration
	zero                 Value

	buckets       []counterBucket
	first, active int

	total Value
	count uint64
}

type counterBucket struct {
	startTime       time.Time
	total, min, max Value
	count           uint64
}

func (b *counterBucket) reset(t time.Time) {
	b.startTime = t
	b.count = 0
	b.total = nil
	b.min = nil
	b.max = nil
}

func (b *counterBucket) add(val Value) {
	if b.count == 0 {
		b.total = val
		b.min = val
		b.max = val
	} else {
		b.total = b.total.Add(val)
		b.min = b.min.Min(val)
		b.max = b.max.Max(val)
	}
	b.count++
}

func NewMovingCounter(clock Clock, period time.Duration, numBuckets int, zero Value) *MovingCounter {
	if zero == nil {
		panic("zero-value must not be nil")
	}
	if clock == nil {
		clock = defaultClock{}
	}
	return &MovingCounter{
		clock:        clock,
		period:       period,
		bucketPeriod: period / time.Duration(numBuckets),
		zero:         zero,
		buckets:      make([]counterBucket, numBuckets),
		total:        zero,
	}
}

func (c *MovingCounter) expireOld(now time.Time) {
	firstTime := now.Add(-c.period)
	for c.active > 0 {
		bucket := c.buckets[c.first]
		if bucket.startTime.After(firstTime) {
			break
		}
		if bucket.count > 0 {
			c.count -= bucket.count
			c.total = c.total.Sub(bucket.total)
		}
		bucket.reset(time.Time{})
		c.active--
		c.first = (c.first + 1) % len(c.buckets)
	}

	if c.active == 0 {
		// Expired everything.
		c.first = 0
		if c.count != 0 {
			panic("c.count != 0")
		}
	}

	// TODO: If the counter value is a float, rounding errors will accumulate
	// over time. We need to occasionally recalculate the total to minimise those
	// rounding errors.
	if c.count == 0 {
		c.total = c.zero
	} else if c.count < 0 {
		panic("c.count < 0")
	}
}

func (c *MovingCounter) getBucket(now time.Time) *counterBucket {
	if c.active > 0 {
		currIndex := (c.first + c.active - 1) % len(c.buckets)
		bucket := &c.buckets[currIndex]
		if now.Before(bucket.startTime) {
			// Going backwards in time.
			return nil
		} else if now.Before(bucket.startTime.Add(c.bucketPeriod)) {
			return bucket
		}
	}

	c.expireOld(now)
	if c.active >= len(c.buckets) {
		panic("c.active >= len(c.buckets)")
	}
	c.active++
	index := (c.first + c.active - 1) % len(c.buckets)
	bucket := &c.buckets[index]
	ut := now.UnixNano()
	st := ut - (ut % c.bucketPeriod.Nanoseconds())
	bucket.reset(time.Unix(0, st))

	return bucket
}

func (c *MovingCounter) iterate(f func(*counterBucket)) {
	for i := 0; i < c.active; i++ {
		index := (c.first + i) % len(c.buckets)
		b := &c.buckets[index]
		if b.count == 0 {
			continue
		}
		f(b)
	}
}

func (c *MovingCounter) Add(val Value) {
	now := c.clock.Now()
	c.addWithTime(val, now)
}

func (c *MovingCounter) addWithTime(val Value, now time.Time) {
	b := c.getBucket(now)
	if b == nil {
		return
	}
	b.add(val)
	c.total = c.total.Add(val)
	c.count++
}

func (c *MovingCounter) Total() (Value, uint64) {
	// Advances time forward and expires old buckets.
	c.expireOld(c.clock.Now())
	return c.total, c.count
}

func (c *MovingCounter) Min() Value {
	// Advances time forward and expires old buckets.
	c.expireOld(c.clock.Now())
	min := c.zero
	hasVal := false
	c.iterate(func(b *counterBucket) {
		if !hasVal {
			min = b.min
			hasVal = true
		} else {
			min = min.Min(b.min)
		}
	})
	return min
}

func (c *MovingCounter) Max() Value {
	// Advances time forward and expires old buckets.
	c.expireOld(c.clock.Now())
	max := c.zero
	hasVal := false
	c.iterate(func(b *counterBucket) {
		if !hasVal {
			max = b.max
			hasVal = true
		} else {
			max = max.Max(b.max)
		}
	})
	return max
}
