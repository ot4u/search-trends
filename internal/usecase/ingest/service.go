package ingest

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ot4/search-trends/internal/domain/trending"
	"github.com/ot4/search-trends/pkg/logger"
)

type AckFunc func() error

var ErrServiceClosed = errors.New("ingest service is closed")

type queuedEvent struct {
	event trending.Event
	ack   AckFunc
}

type Service struct {
	logger    logger.Logger
	processor *Processor
	queue     chan queuedEvent
	tickEvery time.Duration
	now       func() time.Time
	onEnqueue func(queueSize int)

	mu     sync.RWMutex
	closed bool
	wg     sync.WaitGroup
}

type ServiceConfig struct {
	Logger    logger.Logger
	Processor *Processor
	QueueSize int
	TickEvery time.Duration
	Now       func() time.Time
	OnEnqueue func(queueSize int)
}

func NewService(cfg ServiceConfig) *Service {
	if cfg.Logger == nil {
		cfg.Logger = logger.Nop()
	}
	if cfg.Processor == nil {
		cfg.Processor = NewProcessor(ProcessorConfig{})
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 8192
	}
	if cfg.TickEvery <= 0 {
		cfg.TickEvery = time.Second
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}

	return &Service{
		logger:    cfg.Logger,
		processor: cfg.Processor,
		queue:     make(chan queuedEvent, cfg.QueueSize),
		tickEvery: cfg.TickEvery,
		now:       cfg.Now,
		onEnqueue: cfg.OnEnqueue,
	}
}

func (s *Service) Start() {
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(s.tickEvery)
		defer ticker.Stop()

		for {
			select {
			case item, ok := <-s.queue:
				if !ok {
					s.processor.Tick(s.now())
					s.logger.Info("ingest writer stopped", logger.Int("queue_size", len(s.queue)))
					return
				}

				result := s.processor.ProcessAt(item.event, s.now())
				if result.Accepted && item.ack != nil {
					if err := item.ack(); err != nil {
						s.logger.Warn(
							"failed to acknowledge processed event",
							logger.Err(err),
							logger.String("request_id", item.event.RequestID),
						)
					}
				}
			case tickAt := <-ticker.C:
				s.processor.Tick(tickAt.UTC())
			}
		}
	}()
}

func (s *Service) Enqueue(ctx context.Context, event trending.Event, ack AckFunc) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return ErrServiceClosed
	}

	select {
	case s.queue <- queuedEvent{event: event, ack: ack}:
		if s.onEnqueue != nil {
			s.onEnqueue(len(s.queue))
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) CloseInput() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	close(s.queue)
}

func (s *Service) Wait() {
	s.wg.Wait()
}

func (s *Service) QueueSize() int {
	return len(s.queue)
}
