package scheduler

import (
	"context"
	"log/slog"
	"time"
)

const maxTick = 10 * time.Minute

type Poster interface {
	PostDueInstallments(ctx context.Context) (int, error)
}

type Installments struct {
	poster Poster
	every  time.Duration
	log    *slog.Logger
}

func NewInstallments(poster Poster, every time.Duration, log *slog.Logger) *Installments {
	return &Installments{poster: poster, every: every, log: log}
}

func (s *Installments) Run(ctx context.Context) {
	if s.every <= 0 {
		s.log.Info("installment scheduler disabled")
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

func (s *Installments) tick(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, min(s.every, maxTick))
	defer cancel()
	posted, err := s.poster.PostDueInstallments(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.log.Error("post due installments", slog.String("error", err.Error()))
		}
		return
	}
	if posted > 0 {
		s.log.Info("installment payments posted", slog.Int("count", posted))
	}
}
