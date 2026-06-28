// errors.go — sentinel errors for the review package.
package review

import "errors"

var (
	ErrReviewNotFound  = errors.New("review not found")
	ErrNotPurchased    = errors.New("you can only review products you have purchased")
	ErrAlreadyReviewed = errors.New("you have already reviewed this product")
	ErrProductNotFound = errors.New("product not found")
)
