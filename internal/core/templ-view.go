package core

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
)

type TemplView struct {
	ctx context.Context
	w   http.ResponseWriter
	r   *http.Request
}

func NewTemplView(ctx context.Context, w http.ResponseWriter, r *http.Request) *TemplView {
	return &TemplView{ctx: ctx, w: w, r: r}
}

func (v *TemplView) Render(c templ.Component) error {
	v.setupHeaders(false)
	return c.Render(v.ctx, v.w)
}

func (v *TemplView) setupHeaders(cache bool) {
	v.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if cache {
		v.w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	} else {
		// No caching headers
		v.w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		v.w.Header().Set("Pragma", "no-cache")
		v.w.Header().Set("Expires", "0")
	}
}
