package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Search(ctx context.Context, params SearchParams) (*SearchResult, error)
	Suggest(ctx context.Context, query string, limit int) ([]Suggestion, error)
	GetByID(ctx context.Context, id string) (*ProductDetail, error)
	Create(ctx context.Context, p Product) (*Product, error)
	Update(ctx context.Context, id string, p UpdateProductRequest) (*Product, error)
	Delete(ctx context.Context, id string) error
	AddImage(ctx context.Context, productID string, url string, isPrimary bool) (*ProductImage, error)
	DeleteImage(ctx context.Context, productID string, imageID string) error
	GetImages(ctx context.Context, productID string) ([]ProductImage, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
	// Build the WHERE clause dynamically.
	var conditions []string
	var args []interface{}
	argIdx := 1

	hasQuery := params.Query != ""
	if hasQuery {
		// Convert user input to a tsquery-safe format: split words and join with &.
		words := strings.Fields(params.Query)
		tsquery := strings.Join(words, " & ")
		conditions = append(conditions, fmt.Sprintf("p.search_vector @@ to_tsquery('english', $%d)", argIdx))
		args = append(args, tsquery)
		argIdx++
	}

	if params.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("p.category_id = $%d", argIdx))
		args = append(args, *params.CategoryID)
		argIdx++
	}
	if params.BrandID != nil {
		conditions = append(conditions, fmt.Sprintf("p.brand_id = $%d", argIdx))
		args = append(args, *params.BrandID)
		argIdx++
	}
	if params.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("p.price >= $%d", argIdx))
		args = append(args, *params.MinPrice)
		argIdx++
	}
	if params.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("p.price <= $%d", argIdx))
		args = append(args, *params.MaxPrice)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Rating filter goes in HAVING since it's an aggregate.
	havingClause := ""
	if params.MinRating != nil {
		havingClause = fmt.Sprintf("HAVING COALESCE(AVG(rv.rating), 0) >= $%d", argIdx)
		args = append(args, *params.MinRating)
		argIdx++
	}

	// Order by clause.
	var orderClause string
	switch params.SortBy {
	case "price_asc":
		orderClause = "ORDER BY p.price ASC"
	case "price_desc":
		orderClause = "ORDER BY p.price DESC"
	case "rating":
		orderClause = "ORDER BY avg_rating DESC"
	default: // "relevance" or empty
		if hasQuery {
			orderClause = "ORDER BY relevance DESC"
		} else {
			orderClause = "ORDER BY p.created_at DESC"
		}
	}

	// Relevance column.
	relevanceCol := "NULL::float8 AS relevance"
	if hasQuery {
		relevanceCol = fmt.Sprintf("ts_rank(p.search_vector, to_tsquery('english', $1)) AS relevance")
	}

	// Primary image subquery.
	primaryImageSub := `(SELECT pi.url FROM product_images pi WHERE pi.product_id = p.id AND pi.is_primary = true LIMIT 1)`

	offset := (params.Page - 1) * params.PageSize

	// Count query (for pagination).
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT p.id
			FROM products p
			LEFT JOIN reviews rv ON rv.product_id = p.id
			%s
			GROUP BY p.id
			%s
		) sub`, whereClause, havingClause)

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count products: %w", err)
	}

	// Main query.
	dataQuery := fmt.Sprintf(`
		SELECT
			p.id, p.name, p.price, p.stock_quantity,
			c.name AS category_name,
			b.name AS brand_name,
			COALESCE(AVG(rv.rating), 0) AS avg_rating,
			COUNT(rv.id) AS review_count,
			%s AS primary_image,
			%s
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN brands b ON p.brand_id = b.id
		LEFT JOIN reviews rv ON rv.product_id = p.id
		%s
		GROUP BY p.id, c.name, b.name
		%s
		%s
		LIMIT $%d OFFSET $%d`,
		primaryImageSub, relevanceCol, whereClause, havingClause, orderClause, argIdx, argIdx+1)

	dataArgs := append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}
	defer rows.Close()

	var products []ProductListItem
	for rows.Next() {
		var item ProductListItem
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Price, &item.StockQuantity,
			&item.CategoryName, &item.BrandName,
			&item.AvgRating, &item.ReviewCount,
			&item.PrimaryImage, &item.Relevance,
		); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, item)
	}
	if products == nil {
		products = []ProductListItem{}
	}

	return &SearchResult{
		Products: products,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

func (r *postgresRepository) Suggest(ctx context.Context, query string, limit int) ([]Suggestion, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name FROM products WHERE name ILIKE '%' || $1 || '%' ORDER BY name LIMIT $2`,
		query, limit)
	if err != nil {
		return nil, fmt.Errorf("suggest products: %w", err)
	}
	defer rows.Close()

	var suggestions []Suggestion
	for rows.Next() {
		var s Suggestion
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, fmt.Errorf("scan suggestion: %w", err)
		}
		suggestions = append(suggestions, s)
	}
	if suggestions == nil {
		suggestions = []Suggestion{}
	}
	return suggestions, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*ProductDetail, error) {
	query := `
		SELECT
			p.id, p.name, p.description, p.price, p.stock_quantity,
			p.category_id, p.brand_id,
			p.weight_g, p.weight_oz, p.dimensions_cm, p.dimensions_inch,
			p.created_at, p.updated_at,
			c.name, c.slug,
			b.name,
			COALESCE(AVG(rv.rating), 0),
			COUNT(rv.id)
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN brands b ON p.brand_id = b.id
		LEFT JOIN reviews rv ON rv.product_id = p.id
		WHERE p.id = $1
		GROUP BY p.id, c.name, c.slug, b.name`

	var d ProductDetail
	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.Name, &d.Description, &d.Price, &d.StockQuantity,
		&d.CategoryID, &d.BrandID,
		&d.WeightG, &d.WeightOz, &d.DimensionsCm, &d.DimensionsInch,
		&d.CreatedAt, &d.UpdatedAt,
		&d.CategoryName, &d.CategorySlug,
		&d.BrandName,
		&d.AvgRating, &d.ReviewCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("get product: %w", err)
	}

	images, err := r.GetImages(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Images = images

	return &d, nil
}

func (r *postgresRepository) Create(ctx context.Context, p Product) (*Product, error) {
	query := `
		INSERT INTO products (name, description, price, stock_quantity, category_id, brand_id, weight_g, dimensions_cm)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, description, price, stock_quantity, category_id, brand_id,
			weight_g, weight_oz, dimensions_cm, dimensions_inch, created_at, updated_at`

	var created Product
	err := r.db.QueryRow(ctx, query,
		p.Name, p.Description, p.Price, p.StockQuantity,
		p.CategoryID, p.BrandID, p.WeightG, p.DimensionsCm,
	).Scan(
		&created.ID, &created.Name, &created.Description, &created.Price, &created.StockQuantity,
		&created.CategoryID, &created.BrandID,
		&created.WeightG, &created.WeightOz, &created.DimensionsCm, &created.DimensionsInch,
		&created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	return &created, nil
}

func (r *postgresRepository) Update(ctx context.Context, id string, req UpdateProductRequest) (*Product, error) {
	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Price != nil {
		setClauses = append(setClauses, fmt.Sprintf("price = $%d", argIdx))
		args = append(args, *req.Price)
		argIdx++
	}
	if req.StockQuantity != nil {
		setClauses = append(setClauses, fmt.Sprintf("stock_quantity = $%d", argIdx))
		args = append(args, *req.StockQuantity)
		argIdx++
	}
	if req.CategoryID != nil {
		setClauses = append(setClauses, fmt.Sprintf("category_id = $%d", argIdx))
		args = append(args, *req.CategoryID)
		argIdx++
	}
	if req.BrandID != nil {
		setClauses = append(setClauses, fmt.Sprintf("brand_id = $%d", argIdx))
		args = append(args, *req.BrandID)
		argIdx++
	}
	if req.WeightG != nil {
		setClauses = append(setClauses, fmt.Sprintf("weight_g = $%d", argIdx))
		args = append(args, *req.WeightG)
		argIdx++
	}
	if req.DimensionsCm != nil {
		setClauses = append(setClauses, fmt.Sprintf("dimensions_cm = $%d", argIdx))
		args = append(args, *req.DimensionsCm)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE products SET %s WHERE id = $%d
		RETURNING id, name, description, price, stock_quantity, category_id, brand_id,
			weight_g, weight_oz, dimensions_cm, dimensions_inch, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx)

	var p Product
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&p.ID, &p.Name, &p.Description, &p.Price, &p.StockQuantity,
		&p.CategoryID, &p.BrandID,
		&p.WeightG, &p.WeightOz, &p.DimensionsCm, &p.DimensionsInch,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("update product: %w", err)
	}
	return &p, nil
}

func (r *postgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r *postgresRepository) AddImage(ctx context.Context, productID string, url string, isPrimary bool) (*ProductImage, error) {
	// If this is the primary image, unset any existing primary first.
	if isPrimary {
		_, err := r.db.Exec(ctx,
			`UPDATE product_images SET is_primary = false WHERE product_id = $1 AND is_primary = true`,
			productID)
		if err != nil {
			return nil, fmt.Errorf("unset primary image: %w", err)
		}
	}

	query := `INSERT INTO product_images (product_id, url, is_primary) VALUES ($1, $2, $3)
		RETURNING id, product_id, url, is_primary`
	var img ProductImage
	err := r.db.QueryRow(ctx, query, productID, url, isPrimary).Scan(
		&img.ID, &img.ProductID, &img.URL, &img.IsPrimary,
	)
	if err != nil {
		return nil, fmt.Errorf("add image: %w", err)
	}
	return &img, nil
}

func (r *postgresRepository) DeleteImage(ctx context.Context, productID string, imageID string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM product_images WHERE id = $1 AND product_id = $2`, imageID, productID)
	if err != nil {
		return fmt.Errorf("delete image: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrImageNotFound
	}
	return nil
}

func (r *postgresRepository) GetImages(ctx context.Context, productID string) ([]ProductImage, error) {
	query := `SELECT id, product_id, url, is_primary FROM product_images
		WHERE product_id = $1 ORDER BY is_primary DESC, created_at`
	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("get images: %w", err)
	}
	defer rows.Close()

	var images []ProductImage
	for rows.Next() {
		var img ProductImage
		if err := rows.Scan(&img.ID, &img.ProductID, &img.URL, &img.IsPrimary); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}
		images = append(images, img)
	}
	if images == nil {
		images = []ProductImage{}
	}
	return images, nil
}
