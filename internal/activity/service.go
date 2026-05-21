package activity

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Logger is a non-blocking facade around the repository. Activity logging is
// best-effort — it must never break the request path. Failures are logged
// and swallowed.
type Logger interface {
	LogView(ctx context.Context, userID, guestID *uuid.UUID, productID, categoryID *uuid.UUID)
	LogSearch(ctx context.Context, userID, guestID *uuid.UUID, query string)
	LogAddToCart(ctx context.Context, userID, guestID *uuid.UUID, productID, categoryID *uuid.UUID)
	LogPurchase(ctx context.Context, userID, guestID *uuid.UUID, productID, categoryID *uuid.UUID)
}

type Reader interface {
	Recent(ctx context.Context, userID, guestID *uuid.UUID, since time.Time, limit int) ([]Event, error)
}

type Service interface {
	Logger
	Reader
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) LogView(ctx context.Context, userID, guestID *uuid.UUID, productID, categoryID *uuid.UUID) {
	s.record(ctx, Event{
		UserID: userID, GuestSessionID: guestID,
		EventType: EventView,
		ProductID: productID, CategoryID: categoryID,
	})
}

func (s *service) LogSearch(ctx context.Context, userID, guestID *uuid.UUID, query string) {
	q := strings.TrimSpace(query)
	if q == "" {
		return
	}
	if len(q) > 200 {
		q = q[:200]
	}
	s.record(ctx, Event{
		UserID: userID, GuestSessionID: guestID,
		EventType:   EventSearch,
		SearchQuery: &q,
	})
}

func (s *service) LogAddToCart(ctx context.Context, userID, guestID *uuid.UUID, productID, categoryID *uuid.UUID) {
	s.record(ctx, Event{
		UserID: userID, GuestSessionID: guestID,
		EventType: EventAddToCart,
		ProductID: productID, CategoryID: categoryID,
	})
}

func (s *service) LogPurchase(ctx context.Context, userID, guestID *uuid.UUID, productID, categoryID *uuid.UUID) {
	s.record(ctx, Event{
		UserID: userID, GuestSessionID: guestID,
		EventType: EventPurchase,
		ProductID: productID, CategoryID: categoryID,
	})
}

func (s *service) Recent(ctx context.Context, userID, guestID *uuid.UUID, since time.Time, limit int) ([]Event, error) {
	return s.repo.Recent(ctx, userID, guestID, since, limit)
}

func (s *service) record(ctx context.Context, evt Event) {
	if evt.UserID == nil && evt.GuestSessionID == nil {
		return
	}
	if err := s.repo.Record(ctx, evt); err != nil {
		log.Printf("activity: record %s failed: %v", evt.EventType, err)
	}
}

// NoopLogger satisfies Logger without doing anything. Useful for tests and
// for the early-startup window before the activity service is wired.
type NoopLogger struct{}

func (NoopLogger) LogView(context.Context, *uuid.UUID, *uuid.UUID, *uuid.UUID, *uuid.UUID)      {}
func (NoopLogger) LogSearch(context.Context, *uuid.UUID, *uuid.UUID, string)                    {}
func (NoopLogger) LogAddToCart(context.Context, *uuid.UUID, *uuid.UUID, *uuid.UUID, *uuid.UUID) {}
func (NoopLogger) LogPurchase(context.Context, *uuid.UUID, *uuid.UUID, *uuid.UUID, *uuid.UUID)  {}
