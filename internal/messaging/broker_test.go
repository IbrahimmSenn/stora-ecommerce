package messaging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBroker_PublishWhileDisconnected(t *testing.T) {
	// A broker with no live connection (mid-reconnect) rejects publishes so the
	// caller can surface the failure and let the provider retry, rather than
	// blocking or panicking on a nil channel.
	b := &Broker{done: make(chan struct{})}
	err := b.Publish(context.Background(), "ex", "rk", []byte("{}"))
	assert.ErrorIs(t, err, ErrNotConnected)
	assert.False(t, b.IsConnected())
}
