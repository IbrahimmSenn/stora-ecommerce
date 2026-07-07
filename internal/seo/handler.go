// handler.go — serves /sitemap.xml and /robots.txt.
package seo

import (
	"encoding/xml"
	"log"
	"net/http"
	"strings"
)

type Handler struct {
	service Service
	baseURL string
}

func NewHandler(service Service, baseURL string) *Handler {
	return &Handler{service: service, baseURL: strings.TrimRight(baseURL, "/")}
}

type xmlURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlset struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []xmlURL `xml:"url"`
}

func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	urls, err := h.service.SitemapURLs(r.Context())
	if err != nil {
		log.Printf("sitemap: %v", err)
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}

	set := urlset{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, u := range urls {
		set.URLs = append(set.URLs, xmlURL{Loc: u.Loc, LastMod: u.LastMod})
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(set); err != nil {
		log.Printf("sitemap encode: %v", err)
	}
}

// Robots overrides the static web/public/robots.txt so the Sitemap directive
// can carry the absolute URL the spec requires.
func (h *Handler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(`User-agent: *
Disallow: /admin
Disallow: /account
Disallow: /checkout
Disallow: /cart
Disallow: /orders
Disallow: /api/

Sitemap: ` + h.baseURL + "/sitemap.xml\n"))
}
