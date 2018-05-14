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

	buckets    []counterBucket
	firstIndex int64
	currIndex  int64

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
	nowIndex := now.UnixNano() / c.bucketPeriod.Nanoseconds()
	expectedFirst := nowIndex - int64(len(c.buckets)) + 1
	for ; c.firstIndex <= c.currIndex && c.firstIndex < expectedFirst; c.firstIndex++ {
		bucketIndex := int(c.firstIndex % int64(len(c.buckets)))
		bucket := &c.buckets[bucketIndex]
		if bucket.startTime.IsZero() {
			continue
		}
		if bucket.count > 0 {
			c.count -= bucket.count
			c.total = c.total.Sub(bucket.total)
		}
		bucket.reset(time.Time{})
	}
	if c.firstIndex > c.currIndex {
		// Expired everything.
		c.firstIndex = c.currIndex
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
	bucketIndex := int(c.currIndex % int64(len(c.buckets)))
	bucket := &c.buckets[bucketIndex]
	diff := now.Sub(bucket.startTime)
	if !bucket.startTime.IsZero() && diff >= 0 && diff < c.bucketPeriod {
		return bucket
	}

	bucketStartTime := now.Truncate(c.bucketPeriod)
	timeIndex := now.UnixNano() / c.bucketPeriod.Nanoseconds()
	if timeIndex < c.firstIndex {
		// Bucket before the current time window.
		return nil
	} else if timeIndex <= c.currIndex {
		// Bucket within the current time window.
		bucketIndex = int(timeIndex % int64(len(c.buckets)))
		return &c.buckets[bucketIndex]
	}

	c.expireOld(now)

	c.currIndex = timeIndex
	newFirst := timeIndex - int64(len(c.buckets)) + 1
	if c.firstIndex < newFirst {
		c.firstIndex = newFirst
	}

	bucketIndex = int(c.currIndex % int64(len(c.buckets)))
	bucket = &c.buckets[bucketIndex]
	bucket.reset(bucketStartTime)

	return bucket
}

func (c *MovingCounter) iterate(now time.Time, f func(*counterBucket)) {
	for i := c.firstIndex; i <= c.currIndex; i++ {
		bucketIndex := int(i % int64(len(c.buckets)))
		b := &c.buckets[bucketIndex]
		if b.startTime.IsZero() || b.count == 0 {
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
	c.iterate(c.clock.Now(), func(b *counterBucket) {
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
	c.iterate(c.clock.Now(), func(b *counterBucket) {
		if !hasVal {
			max = b.max
			hasVal = true
		} else {
			max = max.Max(b.max)
		}
	})
	return max
}
