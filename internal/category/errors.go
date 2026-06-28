// errors.go — sentinel errors for the category package.
package category

import "errors"

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryExists   = errors.New("category already exists")
	// ErrCategoryInUse is returned when a delete is blocked because the category
	// still has child categories or products pointing at it.
	ErrCategoryInUse = errors.New("category is in use")
)
