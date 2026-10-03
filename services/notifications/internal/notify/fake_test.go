package notify

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nnc/family-manager/services/notifications/internal/expo"
)

type fakeSender struct {
	mu       sync.Mutex
	batches  [][]expo.Message
	dead     map[string]bool
	fail     error
	receipts map[string]expo.Receipt
	asked    [][]string
	seq      int
}

func newFakeSender() *fakeSender {
	return &fakeSender{dead: map[string]bool{}, receipts: map[string]expo.Receipt{}}
}

func (s *fakeSender) Send(_ context.Context, msgs []expo.Message) ([]expo.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail != nil {
		return nil, s.fail
	}
	if len(msgs) > expo.MaxBatch {
		return nil, errors.New("batch over the expo limit")
	}
	s.batches = append(s.batches, msgs)
	tickets := make([]expo.Ticket, len(msgs))
	for i, m := range msgs {
		if s.dead[m.To] {
			tickets[i] = expo.Ticket{Status: expo.StatusError, Error: expo.DeviceNotRegistered}
			continue
		}
		s.seq++
		tickets[i] = expo.Ticket{Status: expo.StatusOK, ID: fmt.Sprintf("ticket-%d", s.seq)}
	}
	return tickets, nil
}

func (s *fakeSender) Receipts(_ context.Context, ids []string) (map[string]expo.Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.asked = append(s.asked, ids)
	out := map[string]expo.Receipt{}
	for _, id := range ids {
		if r, ok := s.receipts[id]; ok {
			out[id] = r
		}
	}
	return out, nil
}

func (s *fakeSender) sent() []expo.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []expo.Message
	for _, b := range s.batches {
		out = append(out, b...)
	}
	return out
}

type fakeFamilies struct {
	of   map[string]string
	fail error
}

func (f *fakeFamilies) FamilyOf(_ context.Context, userID string) (string, error) {
	if f.fail != nil {
		return "", f.fail
	}
	return f.of[userID], nil
}
