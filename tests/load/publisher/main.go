package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/ot4/search-trends/internal/domain/trending"
)

func main() {
	var (
		natsURL  = flag.String("nats-url", "nats://127.0.0.1:4222", "NATS server URL")
		subject  = flag.String("subject", "search.events", "JetStream subject")
		count    = flag.Int("count", 50000, "Number of events to publish in burst mode")
		rate     = flag.Int("rate", 0, "Events per second in stream mode")
		duration = flag.Duration("duration", 0, "Duration for stream mode")
	)
	flag.Parse()

	nc, err := natsgo.Connect(*natsURL)
	if err != nil {
		log.Fatalf("connect nats: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("jetstream: %v", err)
	}

	if *rate > 0 && *duration > 0 {
		stream(js, *subject, *rate, *duration)
		return
	}

	burst(js, *subject, *count)
}

func burst(js natsgo.JetStreamContext, subject string, count int) {
	now := time.Now().UTC()

	for i := 0; i < count; i++ {
		if err := publish(js, subject, i, now.Add(-time.Duration(i%30)*time.Second)); err != nil {
			log.Fatalf("publish event: %v", err)
		}
	}

	log.Printf("published %d burst events to %s", count, subject)
}

func stream(js natsgo.JetStreamContext, subject string, rate int, duration time.Duration) {
	interval := time.Second / time.Duration(rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadline := time.Now().Add(duration)
	sent := 0

	for tickAt := range ticker.C {
		if tickAt.After(deadline) {
			break
		}

		if err := publish(js, subject, sent, tickAt.UTC()); err != nil {
			log.Fatalf("publish event: %v", err)
		}
		sent++
	}

	log.Printf("streamed %d events to %s at rate=%d/s for %s", sent, subject, rate, duration)
}

func publish(js natsgo.JetStreamContext, subject string, sequence int, ts time.Time) error {
	event := trending.Event{
		RequestID: fmt.Sprintf("load-%d", sequence),
		Query:     pickQuery(sequence),
		Timestamp: ts.UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = js.Publish(subject, payload)
	return err
}

func pickQuery(i int) string {
	hot := []string{
		"iphone 16",
		"samsung s26",
		"macbook air",
		"airpods pro",
		"nintendo switch 2",
	}

	if i%10 < 7 {
		return hot[i%len(hot)]
	}

	return fmt.Sprintf("tail-query-%d-%d", i, rand.IntN(10000))
}
