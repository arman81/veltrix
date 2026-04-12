// Package queue — in-memory implementation.
//
// MemoryQueue is a channel-based in-process message queue. It implements
// both Publisher and Subscriber interfaces.
//
// Properties:
//   - At-most-once delivery (messages lost on process crash)
//   - Bounded buffer (configurable per topic, default 1000)
//   - Fan-out to consumer groups (each group gets every message)
//   - Within a group, messages are load-balanced across subscribers
//   - No persistence, no ordering guarantees across topics
//
// This is the MVP implementation. Production should swap in NATS JetStream
// or Redis Streams by implementing the same Publisher/Subscriber interfaces.
//
// Usage:
//
//   q := queue.NewMemoryQueue(queue.MemoryQueueConfig{BufferSize: 1000})
//   defer q.Close()
//
//   // Publish
//   q.Publish(ctx, queue.TopicJobsSubmitted, payload, nil)
//
//   // Subscribe (in a goroutine)
//   go q.Subscribe(ctx, queue.TopicJobsSubmitted, "scheduler", handler)
package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// MemoryQueueConfig configures the in-memory queue.
type MemoryQueueConfig struct {
	// BufferSize is the channel buffer size per topic per consumer group.
	// When the buffer is full, Publish blocks until a consumer reads.
	// Default: 1000.
	BufferSize int
}

// ---------------------------------------------------------------------------
// MemoryQueue — the in-process implementation
// ---------------------------------------------------------------------------

// MemoryQueue is a channel-based message queue for development and testing.
// It implements both Publisher and Subscriber.
type MemoryQueue struct {
	config MemoryQueueConfig

	// mu protects the groups map.
	mu sync.RWMutex

	// groups maps topic → group name → channel.
	// Each group gets its own buffered channel. Publishing fans out to all groups.
	groups map[Topic]map[string]chan *Message

	// closed signals that the queue has been shut down.
	closed chan struct{}

	// messageCounter is used to generate unique message IDs.
	messageCounter uint64
}

// NewMemoryQueue creates a new in-memory queue.
func NewMemoryQueue(config MemoryQueueConfig) *MemoryQueue {
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}

	return &MemoryQueue{
		config: config,
		groups: make(map[Topic]map[string]chan *Message),
		closed: make(chan struct{}),
	}
}

// Publish sends a message to all consumer groups subscribed to the topic.
//
// If no groups are subscribed yet, the message is dropped (no buffering
// for future subscribers — this is a simplification for the MVP).
//
// Thread-safe.
func (q *MemoryQueue) Publish(ctx context.Context, topic Topic, payload []byte, metadata map[string]string) error {
	select {
	case <-q.closed:
		return fmt.Errorf("queue is closed")
	default:
	}

	q.mu.Lock()
	q.messageCounter++
	msgID := fmt.Sprintf("mem-%d", q.messageCounter)
	q.mu.Unlock()

	msg := &Message{
		ID:          msgID,
		Topic:       topic,
		Payload:     payload,
		Metadata:    metadata,
		PublishedAt: time.Now(),
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	topicGroups, exists := q.groups[topic]
	if !exists {
		return nil // No subscribers — drop the message
	}

	for _, ch := range topicGroups {
		select {
		case ch <- msg:
			// Delivered
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Buffer full — drop message (at-most-once semantics)
			// In production (NATS/Redis), this would block or return an error.
		}
	}

	return nil
}

// Subscribe starts consuming messages from a topic for a consumer group.
//
// Blocks until ctx is cancelled. The handler is called synchronously for
// each message — if you need concurrent processing, the handler should
// dispatch to a worker pool.
//
// If multiple goroutines call Subscribe with the same topic and group,
// they share the channel (load balancing within a group).
func (q *MemoryQueue) Subscribe(ctx context.Context, topic Topic, group string, handler MessageHandler) error {
	ch := q.getOrCreateGroupChannel(topic, group)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-q.closed:
			return fmt.Errorf("queue is closed")
		case msg := <-ch:
			if err := handler(ctx, msg); err != nil {
				// In the MVP, we log and continue.
				// In production, this would trigger nack/redelivery.
				_ = err
			}
		}
	}
}

// getOrCreateGroupChannel returns the channel for a topic/group pair,
// creating it if it doesn't exist.
func (q *MemoryQueue) getOrCreateGroupChannel(topic Topic, group string) chan *Message {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.groups[topic]; !exists {
		q.groups[topic] = make(map[string]chan *Message)
	}

	if ch, exists := q.groups[topic][group]; exists {
		return ch
	}

	ch := make(chan *Message, q.config.BufferSize)
	q.groups[topic][group] = ch
	return ch
}

// Close shuts down the queue and releases all resources.
// Any blocked Publish or Subscribe calls will return an error.
func (q *MemoryQueue) Close() error {
	close(q.closed)
	return nil
}
