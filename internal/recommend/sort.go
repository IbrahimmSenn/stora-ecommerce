package recommend

import (
	"sort"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/product"
)

type scored struct {
	c     product.Candidate
	score float64
}

// sortScored sorts in-place: score desc, then avg_rating desc as tiebreaker.
func sortScored(items []scored) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].c.AvgRating > items[j].c.AvgRating
	})
}
