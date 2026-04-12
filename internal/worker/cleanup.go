package worker

import (
	"context"
	"log"
	"time"

	"go-cloud-ide/internal/docker"
	"go-cloud-ide/internal/store"
)

// Start periodically stops workspaces that have been inactive past the timeout.
// Workspaces are stopped (not deleted) to preserve data for later restart.
func Start(s *store.Store, d *docker.Client) {
	ticker := time.NewTicker(30 * time.Minute)

	for range ticker.C {
		list, err := s.List()
		if err != nil {
			log.Printf("cleanup skipped: %v", err)
			continue
		}

		for _, ws := range list {
			// Only stop running workspaces that have been inactive
			if ws.Status == store.StatusRunning && time.Since(ws.LastActive) > 30*time.Minute {
				log.Printf("cleanup: stopping inactive workspace=%s (inactive for %v)", ws.ID, time.Since(ws.LastActive))

				if err := d.StopContainer(context.Background(), ws.ContainerID); err != nil {
					log.Printf("cleanup stop failed: workspace=%s err=%v", ws.ID, err)
					continue
				}

				if err := s.UpdateStatus(ws.ID, store.StatusStopped); err != nil {
					log.Printf("cleanup status update failed: workspace=%s err=%v", ws.ID, err)
				}
			}
		}
	}
}
