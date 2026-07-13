// Package recommend produces personalised "you might also like" rails based
// on the shopper's recent activity (views, cart adds, purchases) plus the
// short-term intent encoded in their current cart.
//
// Scoring is intentionally simple and explainable, in the spirit of the rest
// of the codebase — no ML, no async pipelines, just a recency-decayed bag of
// (category, brand) signals over the last 30 days, blended with cart-as-intent
// and a small rating tiebreaker. Diversity is enforced by capping output per
// category so the rail can't collapse onto one taste.
package recommend

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/activity"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/product"
)

// Scoring weights. Kept as package-level constants so the algorithm is one
// place to tune. None of these is sacred — pick numbers that produce results
// you'd actually buy.
const (
	weightView      = 1.0
	weightAddToCart = 3.0
	weightPurchase  = 5.0
	weightCartIntent = 4.0 // applied to the current cart's categories

	// Recency decay: signals lose half their weight every halfLifeDays days.
	halfLifeDays = 7.0

	// Relative pull of each signal in the final score.
	categoryMultiplier = 1.0
	brandMultiplier    = 0.5
	ratingMultiplier   = 0.1

	// Pool size + diversity.
	maxCandidates    = 100
	maxPerCategory   = 2
	signalEventFloor = 3 // below this we fall back to cart-category only
	historyWindow    = 30 * 24 * time.Hour
	maxHistoryEvents = 100
)

type Service interface {
	Recommend(ctx context.Context, userID, guestID *uuid.UUID, cartProductIDs []string, limit int) ([]product.ProductListItem, error)
}

type service struct {
	activity activity.Reader
	products product.Repository
	now      func() time.Time
}

func NewService(activityReader activity.Reader, products product.Repository) Service {
	return &service{
		activity: activityReader,
		products: products,
		now:      time.Now,
	}
}

func (s *service) Recommend(ctx context.Context, userID, guestID *uuid.UUID, cartProductIDs []string, limit int) ([]product.ProductListItem, error) {
	if limit <= 0 {
		return []product.ProductListItem{}, nil
	}

	now := s.now()
	since := now.Add(-historyWindow)

	events, err := s.activity.Recent(ctx, userID, guestID, since, maxHistoryEvents)
	if err != nil {
		return nil, err
	}

	// Build category/brand weight maps from activity. Recency decay is applied
	// per event so a view from yesterday outweighs a view from two weeks ago.
	categoryWeights := map[uuid.UUID]float64{}
	brandWeights := map[uuid.UUID]float64{}
	recentlyViewed := map[uuid.UUID]struct{}{}

	signalEvents := 0
	for _, evt := range events {
		base := signalBase(evt.EventType)
		if base == 0 {
			continue
		}
		decay := decayWeight(now.Sub(evt.OccurredAt))
		w := base * decay

		signalEvents++

		if evt.CategoryID != nil {
			categoryWeights[*evt.CategoryID] += w
		}
		if evt.ProductID != nil && evt.EventType == activity.EventView {
			// Suppress products viewed in the last 24h — the shopper just
			// looked, no need to nag.
			if now.Sub(evt.OccurredAt) < 24*time.Hour {
				recentlyViewed[*evt.ProductID] = struct{}{}
			}
		}
	}

	// Cart-as-intent: load categories/brands of current cart items and add a
	// strong weight to their categories. This is the "people who put X in
	// their cart also like Y in the same category" signal.
	cartCB, err := s.products.CategoryBrandFor(ctx, cartProductIDs)
	if err != nil {
		return nil, err
	}
	for _, cb := range cartCB {
		if cb.CategoryID != nil {
			categoryWeights[*cb.CategoryID] += weightCartIntent
		}
		if cb.BrandID != nil {
			brandWeights[*cb.BrandID] += weightCartIntent * 0.5
		}
	}

	// Cold-start: if the shopper has almost no history AND no cart, there's
	// nothing to personalise against. Return an empty list and let the UI
	// decide whether to hide the rail or show a generic "new arrivals" row.
	if signalEvents < signalEventFloor && len(cartProductIDs) == 0 {
		return []product.ProductListItem{}, nil
	}

	// Exclude cart items and recently-viewed items from the candidate set.
	exclude := append([]string(nil), cartProductIDs...)
	for id := range recentlyViewed {
		exclude = append(exclude, id.String())
	}

	candidates, err := s.products.Candidates(ctx, exclude, maxCandidates)
	if err != nil {
		return nil, err
	}

	ranked := make([]scored, 0, len(candidates))
	for _, c := range candidates {
		score := 0.0
		if c.CategoryID != nil {
			score += categoryMultiplier * categoryWeights[*c.CategoryID]
		}
		if c.BrandID != nil {
			score += brandMultiplier * brandWeights[*c.BrandID]
		}
		score += ratingMultiplier * c.AvgRating

		// Skip products with literally zero affinity in any dimension —
		// otherwise we'd just be returning "newest products with stock".
		if score <= ratingMultiplier*c.AvgRating && c.AvgRating == 0 {
			continue
		}
		ranked = append(ranked, scored{c: c, score: score})
	}

	// Sort by score desc, ties broken by rating desc.
	sortScored(ranked)

	// Diversity cap.
	out := make([]product.ProductListItem, 0, limit)
	perCategory := map[uuid.UUID]int{}
	for _, r := range ranked {
		if len(out) >= limit {
			break
		}
		if r.c.CategoryID != nil {
			if perCategory[*r.c.CategoryID] >= maxPerCategory {
				continue
			}
			perCategory[*r.c.CategoryID]++
		}
		out = append(out, r.c.ProductListItem)
	}
	return out, nil
}

func signalBase(eventType string) float64 {
	switch eventType {
	case activity.EventView:
		return weightView
	case activity.EventAddToCart:
		return weightAddToCart
	case activity.EventPurchase:
		return weightPurchase
	default:
		return 0
	}
}

// decayWeight returns exp(-age_days * ln(2) / halfLifeDays), i.e. an event
// loses half its weight every halfLifeDays.
func decayWeight(age time.Duration) float64 {
	if age < 0 {
		age = 0
	}
	ageDays := age.Hours() / 24
	return math.Exp(-ageDays * math.Ln2 / halfLifeDays)
}
