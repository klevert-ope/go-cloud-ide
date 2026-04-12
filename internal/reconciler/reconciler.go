package reconciler

import (
	"context"
	"log"
	"strconv"
	"strings"

	"go-cloud-ide/internal/docker"
	"go-cloud-ide/internal/store"
)

// Run reconciles the database workspace records with the actual Docker container states.
// This should be called once at server startup.
func Run(s *store.Store, d *docker.Client) error {
	ctx := context.Background()

	// Get all containers from Docker
	containers, err := d.ListContainers(ctx)
	if err != nil {
		return err
	}

	// Build a map of container IDs to their running state
	containerMap := make(map[string]bool)
	containerPortMap := make(map[string]string)
	for _, c := range containers {
		// Filter for workspace containers (they start with "ws-")
		if strings.HasPrefix(c.Names[0], "/ws-") {
			id := c.ID
			// Store running state
			containerMap[id] = c.State == "running"
			// Get port mapping
			if len(c.Ports) > 0 {
				containerPortMap[id] = strconv.Itoa(int(c.Ports[0].PublicPort))
			}
		}
	}

	// Get all workspaces from database
	workspaces, err := s.List()
	if err != nil {
		return err
	}

	for _, ws := range workspaces {
		// Check if container exists
		isRunning, exists := containerMap[ws.ContainerID]

		if !exists {
			// Container is gone, mark workspace as stopped
			if ws.Status != store.StatusStopped {
				log.Printf("Reconciler: Container %s for workspace %s not found, marking as stopped", ws.ContainerID, ws.ID)
				if err := s.UpdateStatus(ws.ID, store.StatusStopped); err != nil {
					log.Printf("Reconciler: Failed to update status for workspace %s: %v", ws.ID, err)
				}
			}
			continue
		}

		// Update status based on container state
		if isRunning && ws.Status != store.StatusRunning {
			log.Printf("Reconciler: Workspace %s container is running, updating status", ws.ID)
			if err := s.UpdateStatus(ws.ID, store.StatusRunning); err != nil {
				log.Printf("Reconciler: Failed to update status for workspace %s: %v", ws.ID, err)
			}
		} else if !isRunning && ws.Status != store.StatusStopped {
			log.Printf("Reconciler: Workspace %s container is stopped, updating status", ws.ID)
			if err := s.UpdateStatus(ws.ID, store.StatusStopped); err != nil {
				log.Printf("Reconciler: Failed to update status for workspace %s: %v", ws.ID, err)
			}
		}
	}

	log.Printf("Reconciler: Synced %d workspaces with Docker state", len(workspaces))
	return nil
}
