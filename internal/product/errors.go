package product

import "errors"

var (
	ErrProductNotFound = errors.New("product not found")
	ErrImageNotFound   = errors.New("product image not found")
)
