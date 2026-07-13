// broker.go — connection supervisor with automatic recovery.
//
// The bare amqp.Connection dies on a broker restart or network blip and never
// comes back on its own: the consumer goroutine would exit and every publish
// would fail until the process restarted. Broker owns the connection, watches
// NotifyClose, and re-dials with backoff — re-declaring topology, re-opening
// the confirm-mode publisher channel, and (via RunConsumer) re-establishing the
// consumer. Publishes during the reconnect window return an error so the caller
// (e.g. the Stripe webhook) can let the provider retry.
package messaging

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ErrNotConnected is returned by Publish while the broker is between connections.
var ErrNotConnected = errors.New("messaging: not connected")

// Broker is a self-healing AMQP connection.
type Broker struct {
	url     string
	declare func(*amqp.Channel) error

	mu    sync.RWMutex
	conn  *amqp.Connection
	pubCh *amqp.Channel

	done chan struct{}
	once sync.Once
}

// NewBroker dials the broker and runs declare (topology) on the publisher
// channel. It returns once an initial connection is established; loss after
// that is recovered in the background.
func NewBroker(ctx context.Context, url string, declare func(*amqp.Channel) error) (*Broker, error) {
	b := &Broker{url: url, declare: declare, done: make(chan struct{})}
	if err := b.connect(ctx); err != nil {
		return nil, err
	}
	return b, nil
}

// connect establishes a fresh connection + confirm-mode publisher channel and
// re-declares topology, then starts a watcher that recovers the next drop.
func (b *Broker) connect(ctx context.Context) error {
	conn, err := Connect(ctx, b.url)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := ch.Confirm(false); err != nil {
		_ = conn.Close()
		return err
	}
	if b.declare != nil {
		if err := b.declare(ch); err != nil {
			_ = conn.Close()
			return err
		}
	}

	b.mu.Lock()
	b.conn = conn
	b.pubCh = ch
	b.mu.Unlock()

	// The watcher must NOT inherit ctx: at boot ctx is the 30s dial deadline,
	// and reconnection has to keep working long after it expires. The watcher's
	// lifetime is bounded by Close() (via b.done), not any request context.
	go b.watch(conn) // #nosec G118 -- reconnect loop is intentionally detached from the boot context
	return nil
}

// watch blocks on the connection's close notification, then reconnects with
// backoff unless the broker is shutting down.
func (b *Broker) watch(conn *amqp.Connection) {
	reason := <-conn.NotifyClose(make(chan *amqp.Error, 1))
	select {
	case <-b.done:
		return // intentional Close()
	default:
	}
	log.Printf("messaging: connection lost (%v) — reconnecting", reason)
	for {
		select {
		case <-b.done:
			return
		case <-time.After(time.Second):
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := b.connect(ctx)
		cancel()
		if err == nil {
			log.Printf("messaging: reconnected")
			return
		}
		log.Printf("messaging: reconnect failed: %v", err)
	}
}

// Publish sends body on the current publisher channel and waits for the broker
// confirm. Returns ErrNotConnected if a reconnect is in progress.
func (b *Broker) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	b.mu.RLock()
	ch := b.pubCh
	conn := b.conn
	b.mu.RUnlock()
	if ch == nil || conn == nil || conn.IsClosed() {
		return ErrNotConnected
	}
	return publishConfirm(ctx, ch, exchange, routingKey, body)
}

// RunConsumer consumes queue until ctx is cancelled, re-establishing the
// consumer whenever its channel or the connection drops. Blocks; run in a
// goroutine.
func (b *Broker) RunConsumer(ctx context.Context, queue string, h Handler) {
	for {
		if ctx.Err() != nil {
			return
		}
		b.mu.RLock()
		conn := b.conn
		b.mu.RUnlock()

		if conn == nil || conn.IsClosed() {
			// Between connections — wait for the watcher to restore one.
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		c, err := NewConsumer(conn, queue)
		if err != nil {
			log.Printf("messaging: open consumer for %s failed: %v", queue, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		err = c.Run(ctx, h) // blocks until ctx done or the channel closes
		_ = c.Close()
		if ctx.Err() != nil {
			return
		}
		log.Printf("messaging: consumer for %s stopped (%v) — re-establishing", queue, err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// IsConnected reports whether the broker currently holds a live connection.
// Used by the health check.
func (b *Broker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.conn != nil && !b.conn.IsClosed()
}

// Close stops recovery and tears down the connection.
func (b *Broker) Close() error {
	b.once.Do(func() { close(b.done) })
	b.mu.RLock()
	conn := b.conn
	b.mu.RUnlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}
