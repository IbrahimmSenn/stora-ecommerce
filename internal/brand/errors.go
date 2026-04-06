// errors.go — sentinel errors for the brand package.
package brand

import "errors"

var (
	ErrBrandNotFound = errors.New("brand not found")
	ErrBrandExists   = errors.New("brand already exists")
)
