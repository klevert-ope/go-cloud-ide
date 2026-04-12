package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"go-cloud-ide/internal/api"
	"go-cloud-ide/internal/apperr"
	"go-cloud-ide/internal/docker"
	"go-cloud-ide/internal/reconciler"
	"go-cloud-ide/internal/store"
	"go-cloud-ide/internal/worker"
)

// main wires the application dependencies and starts the HTTP server.
func main() {
	dbPath := "data/data.db"
	if err := ensureDatabaseFile(dbPath); err != nil {
		log.Fatal(err)
	}

	dockerClient, err := docker.New()
	if err != nil {
		log.Fatal(err)
	}

	workspaceStore, err := store.New(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	h := &api.Handler{Docker: dockerClient, Store: workspaceStore}

	// Reconcile database state with Docker on startup
	if err := reconciler.Run(workspaceStore, dockerClient); err != nil {
		log.Printf("Warning: reconciler failed: %v", err)
	}

	go worker.Start(workspaceStore, dockerClient)

	http.HandleFunc("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			h.Create(w, r)
			return
		}

		if r.Method == "GET" {
			h.List(w, r)
			return
		}

		apperr.Write(w, r, apperr.New("http.workspaces", apperr.KindMethod, "method not allowed"))
	})

	http.HandleFunc("/workspaces/stop", h.Stop)
	http.HandleFunc("/workspaces/start", h.Start)
	http.HandleFunc("/workspaces/restart", h.Restart)
	http.HandleFunc("/", h.UIIndex)
	http.HandleFunc("/ui/workspaces", h.UIWorkspaces)
	http.HandleFunc("/delete", h.Delete)
	http.HandleFunc("/heartbeat", h.Heartbeat)
	http.HandleFunc("/ws/", h.Proxy)

	log.Println("Running on :8090")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		log.Fatal(apperr.E("http.listen", apperr.KindInternal, "server stopped unexpectedly", err))
	}
}

func ensureDatabaseFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.E("main.db.mkdir", apperr.KindInternal, "failed to create database directory", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return apperr.E("main.db.create", apperr.KindInternal, "failed to create database file", err)
	}

	if err := file.Close(); err != nil {
		return apperr.E("main.db.close", apperr.KindInternal, "failed to initialize database file", err)
	}

	return nil
}
