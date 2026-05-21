package activity

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Record(ctx context.Context, evt Event) error
	Recent(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, since time.Time, limit int) ([]Event, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

// Record inserts a single activity event. CHECK constraints in the table
// guarantee well-formed rows; callers should populate ProductID for non-search
// events and SearchQuery for search events.
func (r *postgresRepository) Record(ctx context.Context, evt Event) error {
	if evt.UserID == nil && evt.GuestSessionID == nil {
		return fmt.Errorf("activity: at least one of user_id or guest_session_id required")
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_activity
			(user_id, guest_session_id, event_type, product_id, category_id, search_query)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		evt.UserID, evt.GuestSessionID, evt.EventType,
		evt.ProductID, evt.CategoryID, evt.SearchQuery,
	)
	if err != nil {
		return fmt.Errorf("insert activity: %w", err)
	}
	return nil
}

// Recent returns the most recent events for the owner, newest first. If a
// user_id is supplied we read that owner's history; otherwise we fall back to
// the guest session. We don't union them because callers (the recommender)
// already merged identities at login time.
func (r *postgresRepository) Recent(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, since time.Time, limit int) ([]Event, error) {
	var (
		rows pgx.Rows
		err  error
	)
	switch {
	case userID != nil:
		rows, err = r.db.Query(ctx,
			`SELECT id, user_id, guest_session_id, event_type, product_id, category_id, search_query, occurred_at
			 FROM user_activity
			 WHERE user_id = $1 AND occurred_at >= $2
			 ORDER BY occurred_at DESC
			 LIMIT $3`,
			*userID, since, limit,
		)
	case guestID != nil:
		rows, err = r.db.Query(ctx,
			`SELECT id, user_id, guest_session_id, event_type, product_id, category_id, search_query, occurred_at
			 FROM user_activity
			 WHERE guest_session_id = $1 AND occurred_at >= $2
			 ORDER BY occurred_at DESC
			 LIMIT $3`,
			*guestID, since, limit,
		)
	default:
		return []Event{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query activity: %w", err)
	}
	defer rows.Close()

	out := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.GuestSessionID, &e.EventType,
			&e.ProductID, &e.CategoryID, &e.SearchQuery, &e.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
