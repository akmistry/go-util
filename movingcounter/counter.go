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
	Min(a Value) Value
	Max(a Value) Value
}
type Int64Value int64

func (v Int64Value) Add(a Value) Value {
	return v + a.(Int64Value)
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
	} else if timeIndex > c.currIndex {
		// Advance time forward.
		// TODO: This can be made more efficient.
		c.currIndex = timeIndex
		c.firstIndex = timeIndex - int64(len(c.buckets)) + 1
		firstTime := time.Unix(0, c.firstIndex*c.bucketPeriod.Nanoseconds())
		for i := c.firstIndex; i < c.currIndex; i++ {
			bucketIndex = int(i % int64(len(c.buckets)))
			bucket = &c.buckets[bucketIndex]
			if !bucket.startTime.IsZero() && bucket.startTime.Before(firstTime) {
				bucket.reset(time.Time{})
			}
		}
		bucketIndex = int(c.currIndex % int64(len(c.buckets)))
		bucket = &c.buckets[bucketIndex]
		bucket.reset(bucketStartTime)
	} else {
		bucketIndex = int(timeIndex % int64(len(c.buckets)))
		bucket = &c.buckets[bucketIndex]
	}

	return bucket
}

func (c *MovingCounter) iterate(now time.Time, f func(*counterBucket)) {
	expireTime := now.Add(-c.period)
	for i, b := range c.buckets {
		if b.startTime.IsZero() {
			continue
		} else if b.startTime.Before(expireTime) {
			c.buckets[i].reset(time.Time{})
		} else {
			f(&c.buckets[i])
		}
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
}

func (c *MovingCounter) Total() (total Value, count uint64) {
	total = c.zero
	c.iterate(c.clock.Now(), func(b *counterBucket) {
		if b.count > 0 {
			total = total.Add(b.total)
			count += b.count
		}
	})
	return
}

func (c *MovingCounter) Min() Value {
	min := c.zero
	hasVal := false
	c.iterate(c.clock.Now(), func(b *counterBucket) {
		if b.count == 0 {
			return
		}
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
	max := c.zero
	hasVal := false
	c.iterate(c.clock.Now(), func(b *counterBucket) {
		if b.count == 0 {
			return
		}
		if !hasVal {
			max = b.max
			hasVal = true
		} else {
			max = max.Max(b.max)
		}
	})
	return max
}
