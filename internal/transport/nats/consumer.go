package nats

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/ot4/search-trends/internal/domain/trending"
	"github.com/ot4/search-trends/internal/infrastructure/metrics"
	"github.com/ot4/search-trends/internal/usecase/ingest"
	"github.com/ot4/search-trends/pkg/logger"
)

type Config struct {
	URL           string
	Stream        string
	Subject       string
	Durable       string
	FetchBatch    int
	FetchTimeout  time.Duration
	AckWait       time.Duration
	MaxAckPending int
	AutoProvision bool
}

type Consumer struct {
	cfg      Config
	enqueuer ingest.Enqueuer
	metrics  *metrics.Metrics
	logger   logger.Logger
}

func NewConsumer(cfg Config, enqueuer ingest.Enqueuer, metrics *metrics.Metrics, log logger.Logger) *Consumer {
	if cfg.FetchBatch <= 0 {
		cfg.FetchBatch = 256
	}
	if cfg.FetchTimeout <= 0 {
		cfg.FetchTimeout = time.Second
	}
	if cfg.AckWait <= 0 {
		cfg.AckWait = 30 * time.Second
	}
	if cfg.MaxAckPending <= 0 {
		cfg.MaxAckPending = 20_000
	}
	if log == nil {
		log = logger.Nop()
	}

	return &Consumer{
		cfg:      cfg,
		enqueuer: enqueuer,
		metrics:  metrics,
		logger:   log,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	nc, err := natsgo.Connect(c.cfg.URL, natsgo.Name("search-trends"))
	if err != nil {
		return err
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		return err
	}

	if c.cfg.AutoProvision {
		if err := c.ensureStream(js); err != nil {
			return err
		}
	}

	sub, err := js.PullSubscribe(
		c.cfg.Subject,
		c.cfg.Durable,
		natsgo.BindStream(c.cfg.Stream),
		natsgo.ManualAck(),
		natsgo.AckExplicit(),
		natsgo.AckWait(c.cfg.AckWait),
		natsgo.MaxAckPending(c.cfg.MaxAckPending),
	)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		messages, err := sub.Fetch(c.cfg.FetchBatch, natsgo.MaxWait(c.cfg.FetchTimeout))
		if err != nil {
			if errors.Is(err, natsgo.ErrTimeout) {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}

		for _, message := range messages {
			var event trending.Event
			if err := json.Unmarshal(message.Data, &event); err != nil {
				if c.metrics != nil {
					c.metrics.ObserveNATS("invalid_json")
				}
				c.logger.Warn("invalid search event", logger.Err(err))
				_ = message.Ack()
				continue
			}

			ack := func() error {
				return message.Ack()
			}

			if err := c.enqueuer.Enqueue(ctx, event, ack); err != nil {
				if ctx.Err() != nil || errors.Is(err, ingest.ErrServiceClosed) {
					return ctx.Err()
				}
				if c.metrics != nil {
					c.metrics.ObserveNATS("enqueue_error")
				}
				return err
			}

			if c.metrics != nil {
				c.metrics.ObserveNATS("queued")
				c.metrics.SetQueueSize(c.enqueuer.QueueSize())
			}
		}
	}
}

func (c *Consumer) ensureStream(js natsgo.JetStreamContext) error {
	_, err := js.StreamInfo(c.cfg.Stream)
	if err == nil {
		return nil
	}
	if !errors.Is(err, natsgo.ErrStreamNotFound) {
		return err
	}

	_, err = js.AddStream(&natsgo.StreamConfig{
		Name:      c.cfg.Stream,
		Subjects:  []string{c.cfg.Subject},
		Storage:   natsgo.FileStorage,
		Retention: natsgo.LimitsPolicy,
		MaxAge:    10 * time.Minute,
	})
	return err
}
