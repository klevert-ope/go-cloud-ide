package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go-cloud-ide/internal/apperr"
	"go-cloud-ide/internal/docker"
	"go-cloud-ide/internal/store"
)

type Handler struct {
	Docker *docker.Client
	Store  *store.Store
}

// Create provisions a new workspace and returns either HTML or JSON for it.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	id := uuid.New().String()
	vol := "vol-" + id

	if err := h.Docker.CreateVolume(r.Context(), vol); err != nil {
		apperr.Write(w, r, err)
		return
	}

	cid, port, err := h.Docker.RunWorkspace(r.Context(), "ws-"+id, vol)
	if err != nil {
		apperr.Write(w, r, err)
		return
	}

	ws := &store.Workspace{
		ID:          id,
		ContainerID: cid,
		Volume:      vol,
		Port:        port,
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	if err := h.Store.Save(ws); err != nil {
		_ = h.Docker.StopAndRemove(r.Context(), cid)
		apperr.Write(w, r, err)
		return
	}

	// HTMX request → return HTML
	if r.Header.Get("HX-Request") == "true" {
		list, err := h.Store.List()
		if err != nil {
			apperr.Write(w, r, err)
			return
		}

		if err := templates.ExecuteTemplate(w, "jobs.html", list); err != nil {
			apperr.Write(w, r, apperr.E("api.create.render", apperr.KindInternal, "failed to render workspace list", err))
		}
		return
	}

	// Normal API response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"id":  id,
		"url": "http://localhost:8080/ws/" + id,
	}); err != nil {
		apperr.Write(w, r, apperr.E("api.create.encode", apperr.KindInternal, "failed to encode response", err))
	}
}

// List returns the current set of workspaces as JSON.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.List()
	if err != nil {
		apperr.Write(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		apperr.Write(w, r, apperr.E("api.list.encode", apperr.KindInternal, "failed to encode response", err))
	}
}

// Delete stops a workspace container and removes its record from storage.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		apperr.Write(w, r, apperr.New("api.delete", apperr.KindInvalid, "workspace id is required"))
		return
	}

	ws, err := h.Store.Get(id)
	if err != nil {
		apperr.Write(w, r, err)
		return
	}

	if err := h.Docker.StopAndRemove(r.Context(), ws.ContainerID); err != nil {
		apperr.Write(w, r, err)
		return
	}

	if err := h.Store.Delete(id); err != nil {
		apperr.Write(w, r, err)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		list, err := h.Store.List()
		if err != nil {
			apperr.Write(w, r, err)
			return
		}

		if err := templates.ExecuteTemplate(w, "jobs.html", list); err != nil {
			apperr.Write(w, r, apperr.E("api.delete.render", apperr.KindInternal, "failed to render workspace list", err))
		}
		return
	}

	w.WriteHeader(204)
}

// Heartbeat refreshes the last-active timestamp for a workspace.
func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		apperr.Write(w, r, apperr.New("api.heartbeat", apperr.KindInvalid, "workspace id is required"))
		return
	}

	if err := h.Store.UpdateLastActive(id); err != nil {
		apperr.Write(w, r, err)
		return
	}

	w.WriteHeader(200)
}

// Proxy redirects the browser to the live code-server instance for a workspace.
func (h *Handler) Proxy(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/ws/")
	if id == "" {
		apperr.Write(w, r, apperr.New("api.proxy", apperr.KindInvalid, "workspace id is required"))
		return
	}

	ws, err := h.Store.Get(id)
	if err != nil {
		apperr.Write(w, r, err)
		return
	}

	if err := h.Docker.WaitUntilReady(r.Context(), ws.Port); err != nil {
		apperr.Write(w, r, err)
		return
	}

	target := "http://localhost:" + ws.Port
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}
