package listeners

import (
	"context"
	"net"

	rl "golang.org/x/time/rate"
)

type rateLimitedListener struct {
	net.Listener
	limiter *rl.Limiter

	ctx context.Context
	cf  context.CancelFunc
}

func NewRateLimited(l net.Listener, rate float64, burst int) net.Listener {
	ctx, cf := context.WithCancel(context.Background())
	return &rateLimitedListener{
		Listener: l,
		limiter:  rl.NewLimiter(rl.Limit(rate), burst),
		ctx:      ctx,
		cf:       cf,
	}
}

func (l *rateLimitedListener) Accept() (net.Conn, error) {
	// Pairing a Context with Close allows us to honour net.Listener's contract
	// that a Close should unblock and waiting Accept.
	l.limiter.Wait(l.ctx)
	return l.Listener.Accept()
}

func (l *rateLimitedListener) Close() error {
	defer l.cf()
	return l.Listener.Close()
}
