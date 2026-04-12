package api

import (
	"html/template"
	"net/http"

	"go-cloud-ide/internal/apperr"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

// UIIndex renders the main dashboard page.
func (h *Handler) UIIndex(w http.ResponseWriter, r *http.Request) {
	if err := templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		apperr.Write(w, r, apperr.E("api.ui.index", apperr.KindInternal, "failed to render page", err))
	}
}

// UIWorkspaces renders the workspace list partial for the dashboard.
func (h *Handler) UIWorkspaces(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.List()
	if err != nil {
		apperr.Write(w, r, err)
		return
	}

	if err := templates.ExecuteTemplate(w, "jobs.html", list); err != nil {
		apperr.Write(w, r, apperr.E("api.ui.workspaces", apperr.KindInternal, "failed to render workspace list", err))
	}
}
