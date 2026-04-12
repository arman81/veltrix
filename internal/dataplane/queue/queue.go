// Package queue defines the generic message queue interface for Veltrix.
//
// The queue is the nervous system of the data plane. It decouples control
// plane services so they communicate asynchronously via messages instead of
// direct function calls. This enables:
//
//   - Resilience: if the placement engine is temporarily down, job.scheduled
//     events accumulate in the queue and are processed when it recovers
//   - Scalability: consumers can be scaled independently
//   - Observability: queue depth is a direct measure of system backpressure
//
// Topics (channels) in Veltrix:
//
//   jobs.submitted       API Server    → Scheduler
//   jobs.scheduled       Scheduler     → Placement Engine
//   placements.decided   Placement Eng → Node Agent
//   metrics.ingested     Node Agent    → Feedback Controller
//   jobs.completed       Node Agent    → Scheduler + Feedback Controller
//
// The interface is intentionally simple — Publish and Subscribe. The MVP
// implementation is an in-process Go channel-based queue. Production can
// swap in NATS, Redis Streams, or Kafka by implementing the same interface.
//
// Message format: opaque []byte. Serialization (JSON, protobuf) is the
// caller's responsibility. The queue does not inspect message contents.
package queue

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Topic — a named channel for messages
// ---------------------------------------------------------------------------

// Topic is a named message channel. Publishers send to a topic.
// Subscribers receive from a topic.
type Topic string

const (
	// TopicJobsSubmitted carries new job submissions from the API server
	// to the scheduler.
	TopicJobsSubmitted Topic = "jobs.submitted"

	// TopicJobsScheduled carries scheduled jobs from the scheduler to
	// the placement engine.
	TopicJobsScheduled Topic = "jobs.scheduled"

	// TopicPlacementsDecided carries placement decisions from the placement
	// engine to the node agents.
	TopicPlacementsDecided Topic = "placements.decided"

	// TopicMetricsIngested carries GPU telemetry from the node agents to
	// the feedback controller.
	TopicMetricsIngested Topic = "metrics.ingested"

	// TopicJobsCompleted carries job completion events from the node agents
	// to the scheduler and feedback controller.
	TopicJobsCompleted Topic = "jobs.completed"
)

// ---------------------------------------------------------------------------
// Message — the unit of communication
// ---------------------------------------------------------------------------

// Message is a single message in the queue.
type Message struct {
	// ID is a unique identifier for this message (for deduplication).
	ID string

	// Topic is the channel this message was published to.
	Topic Topic

	// Payload is the serialized message content (JSON, protobuf, etc.).
	// The queue treats this as opaque bytes.
	Payload []byte

	// Metadata is optional key-value pairs attached to the message.
	// Useful for routing, filtering, or tracing without deserializing payload.
	// Example: {"node_id": "node-7", "trace_id": "abc123"}
	Metadata map[string]string

	// PublishedAt is when the message was published.
	PublishedAt time.Time
}

// ---------------------------------------------------------------------------
// Publisher — sends messages to a topic
// ---------------------------------------------------------------------------

// Publisher sends messages to the queue. Implementations must be safe
// for concurrent use from multiple goroutines.
type Publisher interface {
	// Publish sends a message to the specified topic.
	//
	// The message ID is set by the publisher if empty.
	// Returns an error if the queue is full or unavailable.
	//
	// Delivery guarantee depends on the implementation:
	//   - In-memory: at-most-once (lost on process crash)
	//   - NATS: at-least-once (with JetStream)
	//   - Redis Streams: at-least-once
	Publish(ctx context.Context, topic Topic, payload []byte, metadata map[string]string) error

	// Close releases any resources held by the publisher.
	Close() error
}

// ---------------------------------------------------------------------------
// Subscriber — receives messages from a topic
// ---------------------------------------------------------------------------

// MessageHandler is a callback function invoked for each received message.
// Return nil to acknowledge the message (remove from queue).
// Return an error to nack the message (redelivery depends on implementation).
type MessageHandler func(ctx context.Context, msg *Message) error

// Subscriber receives messages from one or more topics. Each subscriber
// belongs to a consumer group — within a group, each message is delivered
// to exactly one subscriber (load balancing). Different groups each get
// a copy of every message (fan-out).
type Subscriber interface {
	// Subscribe registers a handler for messages on the given topic.
	//
	// The handler is called in a goroutine for each message. Multiple
	// messages may be processed concurrently (up to the implementation's
	// concurrency limit).
	//
	// Subscribe blocks until the context is cancelled. It should be called
	// in a goroutine.
	//
	// The group parameter identifies the consumer group. Two subscribers
	// with the same group on the same topic share the message load.
	// Two subscribers with different groups each get all messages.
	Subscribe(ctx context.Context, topic Topic, group string, handler MessageHandler) error

	// Close releases any resources held by the subscriber.
	Close() error
}
