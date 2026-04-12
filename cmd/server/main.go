package main

import (
	"log"
	"net/http"

	"go-cloud-ide/internal/api"
	"go-cloud-ide/internal/apperr"
	"go-cloud-ide/internal/docker"
	"go-cloud-ide/internal/store"
	"go-cloud-ide/internal/worker"
)

// main wires the application dependencies and starts the HTTP server.
func main() {
	dockerClient, err := docker.New()
	if err != nil {
		log.Fatal(err)
	}

	workspaceStore, err := store.New("data/data.db")
	if err != nil {
		log.Fatal(err)
	}

	h := &api.Handler{Docker: dockerClient, Store: workspaceStore}

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

	http.HandleFunc("/", h.UIIndex)
	http.HandleFunc("/ui/workspaces", h.UIWorkspaces)
	http.HandleFunc("/delete", h.Delete)
	http.HandleFunc("/heartbeat", h.Heartbeat)
	http.HandleFunc("/ws/", h.Proxy)

	log.Println("Running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(apperr.E("http.listen", apperr.KindInternal, "server stopped unexpectedly", err))
	}
}
