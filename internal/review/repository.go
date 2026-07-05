// repository.go — postgres queries for reviews, helpful votes, and moderation.
package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

type Repository interface {
	Create(ctx context.Context, userID, productID uuid.UUID, rating int, comment *string) (*Review, error)
	ListByProduct(ctx context.Context, params ListParams) (*ListResult, error)
	HasPurchased(ctx context.Context, userID, productID uuid.UUID) (bool, error)
	GetUserReview(ctx context.Context, userID, productID uuid.UUID) (*Review, error)
	AddVote(ctx context.Context, reviewID, userID uuid.UUID) error
	RemoveVote(ctx context.Context, reviewID, userID uuid.UUID) error

	// Moderation
	ListForModeration(ctx context.Context, status string, page, pageSize int) ([]ModerationItem, int, error)
	UpdateStatus(ctx context.Context, reviewID uuid.UUID, status string) error
	Delete(ctx context.Context, reviewID uuid.UUID) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, userID, productID uuid.UUID, rating int, comment *string) (*Review, error) {
	query := `
		INSERT INTO reviews (user_id, product_id, rating, comment, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, product_id, rating, comment, status, created_at, updated_at`

	var rv Review
	err := r.db.QueryRow(ctx, query, userID, productID, rating, comment, StatusApproved).Scan(
		&rv.ID, &rv.UserID, &rv.ProductID, &rv.Rating, &rv.Comment, &rv.Status,
		&rv.CreatedAt, &rv.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case uniqueViolation:
				return nil, ErrAlreadyReviewed
			case foreignKeyViolation:
				return nil, ErrProductNotFound
			}
		}
		return nil, fmt.Errorf("create review: %w", err)
	}
	return &rv, nil
}

func (r *postgresRepository) ListByProduct(ctx context.Context, params ListParams) (*ListResult, error) {
	var orderClause string
	switch params.Sort {
	case SortNewest:
		orderClause = "rv.created_at DESC"
	case SortHighest:
		orderClause = "rv.rating DESC, rv.created_at DESC"
	case SortLowest:
		orderClause = "rv.rating ASC, rv.created_at DESC"
	default: // SortHelpful
		orderClause = "helpful_count DESC, rv.created_at DESC"
	}

	offset := (params.Page - 1) * params.PageSize

	// viewerID is passed as $4 for the voted-by-me / mine flags; a nil viewer
	// resolves the comparisons to false.
	var viewer interface{}
	if params.ViewerID != nil {
		viewer = *params.ViewerID
	}

	dataQuery := fmt.Sprintf(`
		SELECT
			rv.id, rv.rating, rv.comment, rv.created_at,
			(SELECT COUNT(*) FROM review_votes v WHERE v.review_id = rv.id) AS helpful_count,
			EXISTS (SELECT 1 FROM review_votes v WHERE v.review_id = rv.id AND v.user_id = $4) AS voted_by_me,
			(rv.user_id = $4) AS mine_to_edit
		FROM reviews rv
		WHERE rv.product_id = $1 AND rv.status = '%s'
		ORDER BY %s
		LIMIT $2 OFFSET $3`, StatusApproved, orderClause)

	rows, err := r.db.Query(ctx, dataQuery, params.ProductID, params.PageSize, offset, viewer)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	reviews := []PublicReview{}
	for rows.Next() {
		var pr PublicReview
		if err := rows.Scan(
			&pr.ID, &pr.Rating, &pr.Comment, &pr.CreatedAt,
			&pr.HelpfulCount, &pr.VotedByMe, &pr.MineToEdit,
		); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, pr)
	}

	// Aggregate count, average, and star distribution in one pass.
	aggQuery := `
		SELECT rating, COUNT(*)
		FROM reviews
		WHERE product_id = $1 AND status = $2
		GROUP BY rating`
	aggRows, err := r.db.Query(ctx, aggQuery, params.ProductID, StatusApproved)
	if err != nil {
		return nil, fmt.Errorf("aggregate reviews: %w", err)
	}
	defer aggRows.Close()

	dist := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	total, weighted := 0, 0
	for aggRows.Next() {
		var star, count int
		if err := aggRows.Scan(&star, &count); err != nil {
			return nil, fmt.Errorf("scan aggregate: %w", err)
		}
		dist[star] = count
		total += count
		weighted += star * count
	}

	avg := 0.0
	if total > 0 {
		avg = float64(weighted) / float64(total)
	}

	return &ListResult{
		Reviews:      reviews,
		Total:        total,
		Page:         params.Page,
		PageSize:     params.PageSize,
		AvgRating:    avg,
		Distribution: dist,
	}, nil
}

// HasPurchased reports whether the user has a settled order containing the
// product. Pending-payment and failed orders do not count.
func (r *postgresRepository) HasPurchased(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			WHERE o.user_id = $1
			  AND oi.product_id = $2
			  AND o.status IN ('paid', 'processing', 'shipped', 'delivered')
		)`, userID, productID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check purchase: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) GetUserReview(ctx context.Context, userID, productID uuid.UUID) (*Review, error) {
	var rv Review
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, product_id, rating, comment, status, created_at, updated_at
		FROM reviews WHERE user_id = $1 AND product_id = $2`, userID, productID).Scan(
		&rv.ID, &rv.UserID, &rv.ProductID, &rv.Rating, &rv.Comment, &rv.Status,
		&rv.CreatedAt, &rv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, fmt.Errorf("get user review: %w", err)
	}
	return &rv, nil
}

// AddVote records a helpful vote. Voting is only allowed on approved reviews;
// a duplicate vote is treated as success (idempotent).
func (r *postgresRepository) AddVote(ctx context.Context, reviewID, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO review_votes (review_id, user_id)
		SELECT $1, $2
		WHERE EXISTS (SELECT 1 FROM reviews WHERE id = $1 AND status = $3)
		ON CONFLICT (review_id, user_id) DO NOTHING`,
		reviewID, userID, StatusApproved)
	if err != nil {
		return fmt.Errorf("add vote: %w", err)
	}
	// Zero rows affected means either the review does not exist / is not
	// approved, or the vote already existed. Disambiguate so callers can 404.
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := r.db.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM reviews WHERE id = $1 AND status = $2)`,
			reviewID, StatusApproved).Scan(&exists); err != nil {
			return fmt.Errorf("verify review: %w", err)
		}
		if !exists {
			return ErrReviewNotFound
		}
	}
	return nil
}

func (r *postgresRepository) RemoveVote(ctx context.Context, reviewID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM review_votes WHERE review_id = $1 AND user_id = $2`, reviewID, userID)
	if err != nil {
		return fmt.Errorf("remove vote: %w", err)
	}
	return nil
}

func (r *postgresRepository) ListForModeration(ctx context.Context, status string, page, pageSize int) ([]ModerationItem, int, error) {
	var conds string
	args := []interface{}{}
	if status != "" {
		conds = "WHERE rv.status = $1"
		args = append(args, status)
	}

	countQuery := "SELECT COUNT(*) FROM reviews rv " + conds
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count moderation reviews: %w", err)
	}

	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`
		SELECT rv.id, rv.product_id, p.name, rv.rating, rv.comment, rv.status, rv.created_at
		FROM reviews rv
		JOIN products p ON p.id = rv.product_id
		%s
		ORDER BY rv.created_at DESC
		LIMIT $%d OFFSET $%d`, conds, len(args)+1, len(args)+2)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list moderation reviews: %w", err)
	}
	defer rows.Close()

	items := []ModerationItem{}
	for rows.Next() {
		var it ModerationItem
		if err := rows.Scan(&it.ID, &it.ProductID, &it.ProductName, &it.Rating,
			&it.Comment, &it.Status, &it.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan moderation review: %w", err)
		}
		items = append(items, it)
	}
	return items, total, nil
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, reviewID uuid.UUID, status string) error {
	tag, err := r.db.Exec(ctx, `UPDATE reviews SET status = $2 WHERE id = $1`, reviewID, status)
	if err != nil {
		return fmt.Errorf("update review status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReviewNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, reviewID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM reviews WHERE id = $1`, reviewID)
	if err != nil {
		return fmt.Errorf("delete review: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReviewNotFound
	}
	return nil
}
