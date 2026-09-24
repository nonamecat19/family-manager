package scheduler

import (
	"context"
	"log/slog"
	"time"
)

type SubscriptionPoster interface {
	PostDueSubscriptions(ctx context.Context) (int, error)
}

type Subscriptions struct {
	poster SubscriptionPoster
	every  time.Duration
	log    *slog.Logger
}

func NewSubscriptions(poster SubscriptionPoster, every time.Duration, log *slog.Logger) *Subscriptions {
	return &Subscriptions{poster: poster, every: every, log: log}
}

func (s *Subscriptions) Run(ctx context.Context) {
	if s.every <= 0 {
		s.log.Info("subscription scheduler disabled")
		return
	}
	ticker := time.NewTicker(s.every)
	defer ticker.Stop()
	for {
		s.tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Subscriptions) tick(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, min(s.every, maxSchedulerTick))
	defer cancel()
	posted, err := s.poster.PostDueSubscriptions(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.log.Error("post due subscriptions", slog.String("error", err.Error()))
		}
		return
	}
	if posted > 0 {
		s.log.Info("subscription payments posted", slog.Int("count", posted))
	}
}
