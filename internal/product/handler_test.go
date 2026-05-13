package product

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- API integration tests for product endpoints ---

// mockProductRepo implements Repository with in-memory storage.
type mockProductRepo struct {
	products map[string]*Product
	images   map[string][]ProductImage
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{
		products: make(map[string]*Product),
		images:   make(map[string][]ProductImage),
	}
}

func (m *mockProductRepo) Search(_ context.Context, params SearchParams) (*SearchResult, error) {
	var items []ProductListItem
	for _, p := range m.products {
		items = append(items, ProductListItem{
			ID: p.ID, Name: p.Name, Price: p.Price, StockQuantity: p.StockQuantity,
		})
	}
	return &SearchResult{
		Products: items, Total: len(items), Page: params.Page, PageSize: params.PageSize,
	}, nil
}

func (m *mockProductRepo) Suggest(_ context.Context, query string, limit int) ([]Suggestion, error) {
	var results []Suggestion
	for _, p := range m.products {
		if len(results) >= limit {
			break
		}
		results = append(results, Suggestion{ID: p.ID, Name: p.Name})
	}
	return results, nil
}

func (m *mockProductRepo) GetByID(_ context.Context, id string) (*ProductDetail, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, ErrProductNotFound
	}
	imgs := m.images[id]
	if imgs == nil {
		imgs = []ProductImage{}
	}
	return &ProductDetail{Product: *p, Images: imgs}, nil
}

func (m *mockProductRepo) Create(_ context.Context, p Product) (*Product, error) {
	p.ID = uuid.New()
	m.products[p.ID.String()] = &p
	return &p, nil
}

func (m *mockProductRepo) Update(_ context.Context, id string, req UpdateProductRequest) (*Product, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, ErrProductNotFound
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Price != nil {
		p.Price = *req.Price
	}
	return p, nil
}

func (m *mockProductRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.products[id]; !ok {
		return ErrProductNotFound
	}
	delete(m.products, id)
	return nil
}

func (m *mockProductRepo) AddImage(_ context.Context, productID string, url string, isPrimary bool) (*ProductImage, error) {
	img := ProductImage{ID: uuid.New(), ProductID: uuid.MustParse(productID), URL: url, IsPrimary: isPrimary}
	m.images[productID] = append(m.images[productID], img)
	return &img, nil
}

func (m *mockProductRepo) DeleteImage(_ context.Context, productID string, imageID string) error {
	imgs := m.images[productID]
	for i, img := range imgs {
		if img.ID.String() == imageID {
			m.images[productID] = append(imgs[:i], imgs[i+1:]...)
			return nil
		}
	}
	return ErrImageNotFound
}

func (m *mockProductRepo) GetImages(_ context.Context, productID string) ([]ProductImage, error) {
	imgs := m.images[productID]
	if imgs == nil {
		imgs = []ProductImage{}
	}
	return imgs, nil
}

// --- Helpers ---

func setupProductHandler() (*Handler, *mockProductRepo) {
	repo := newMockProductRepo()
	svc := NewService(repo)
	h := NewHandler(svc)
	return h, repo
}

func seedProduct(repo *mockProductRepo) *Product {
	p := &Product{
		ID: uuid.New(), Name: "Test Laptop", Price: 99900,
		StockQuantity: 10, WeightG: intPtr(2000),
	}
	repo.products[p.ID.String()] = p
	return p
}

// withURLParam creates a request with chi URL params for route matching.
func withURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- Search endpoint tests ---

func TestSearchEndpoint_EmptyStore(t *testing.T) {
	h, _ := setupProductHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result SearchResult
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	assert.Empty(t, result.Products)
	assert.Equal(t, 0, result.Total)
}

func TestSearchEndpoint_WithProducts(t *testing.T) {
	h, repo := setupProductHandler()
	seedProduct(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result SearchResult
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "Test Laptop", result.Products[0].Name)
}

func TestSearchEndpoint_QueryParams(t *testing.T) {
	h, repo := setupProductHandler()
	seedProduct(repo)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/products?q=laptop&min_price=5000&max_price=200000&sort=price_asc&page=1&page_size=10", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- GetByID endpoint tests ---

func TestGetByIDEndpoint_Found(t *testing.T) {
	h, repo := setupProductHandler()
	p := seedProduct(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+p.ID.String(), nil)
	req = withURLParam(req, "id", p.ID.String())
	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var detail ProductDetail
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &detail))
	assert.Equal(t, p.Name, detail.Name)
	assert.Equal(t, p.Price, detail.Price)
}

func TestGetByIDEndpoint_NotFound(t *testing.T) {
	h, _ := setupProductHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+uuid.NewString(), nil)
	req = withURLParam(req, "id", uuid.NewString())
	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// --- Create endpoint tests ---

func TestCreateEndpoint_Success(t *testing.T) {
	h, _ := setupProductHandler()

	body := CreateProductRequest{
		Name: "New Phone", Price: 49900, StockQuantity: 50, WeightG: intPtr(200),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var p Product
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &p))
	assert.Equal(t, "New Phone", p.Name)
	assert.Equal(t, int64(49900), p.Price)
}

func TestCreateEndpoint_MissingName(t *testing.T) {
	h, _ := setupProductHandler()

	body := CreateProductRequest{Price: 49900, WeightG: intPtr(200)}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateEndpoint_InvalidJSON(t *testing.T) {
	h, _ := setupProductHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products",
		bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// --- Update endpoint tests ---

func TestUpdateEndpoint_Success(t *testing.T) {
	h, repo := setupProductHandler()
	p := seedProduct(repo)

	newName := "Updated Laptop"
	body := UpdateProductRequest{Name: &newName}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/products/"+p.ID.String(), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", p.ID.String())
	rr := httptest.NewRecorder()
	h.Update(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var updated Product
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &updated))
	assert.Equal(t, "Updated Laptop", updated.Name)
}

func TestUpdateEndpoint_NotFound(t *testing.T) {
	h, _ := setupProductHandler()

	newName := "Ghost"
	body := UpdateProductRequest{Name: &newName}
	b, _ := json.Marshal(body)
	fakeID := uuid.NewString()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/products/"+fakeID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", fakeID)
	rr := httptest.NewRecorder()
	h.Update(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// --- Delete endpoint tests ---

func TestDeleteEndpoint_Success(t *testing.T) {
	h, repo := setupProductHandler()
	p := seedProduct(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/products/"+p.ID.String(), nil)
	req = withURLParam(req, "id", p.ID.String())
	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDeleteEndpoint_NotFound(t *testing.T) {
	h, _ := setupProductHandler()

	fakeID := uuid.NewString()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/products/"+fakeID, nil)
	req = withURLParam(req, "id", fakeID)
	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// --- Image endpoint tests ---

func TestAddImageEndpoint_Success(t *testing.T) {
	h, repo := setupProductHandler()
	p := seedProduct(repo)

	body := AddImageRequest{URL: "https://example.com/img.jpg", IsPrimary: true}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products/"+p.ID.String()+"/images", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", p.ID.String())
	rr := httptest.NewRecorder()
	h.AddImage(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var img ProductImage
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &img))
	assert.Equal(t, "https://example.com/img.jpg", img.URL)
	assert.True(t, img.IsPrimary)
}

func TestAddImageEndpoint_InvalidURL(t *testing.T) {
	h, repo := setupProductHandler()
	p := seedProduct(repo)

	body := AddImageRequest{URL: "not-a-url", IsPrimary: false}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products/"+p.ID.String()+"/images", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", p.ID.String())
	rr := httptest.NewRecorder()
	h.AddImage(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteImageEndpoint_NotFound(t *testing.T) {
	h, repo := setupProductHandler()
	p := seedProduct(repo)

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/admin/products/"+p.ID.String()+"/images/"+uuid.NewString(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", p.ID.String())
	rctx.URLParams.Add("imageId", uuid.NewString())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.DeleteImage(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
