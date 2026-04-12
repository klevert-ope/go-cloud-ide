package worker

import (
	"context"
	"log"
	"time"

	"go-cloud-ide/internal/docker"
	"go-cloud-ide/internal/store"
)

// Start periodically removes workspaces that have been inactive past the timeout.
func Start(s *store.Store, d *docker.Client) {
	ticker := time.NewTicker(5 * time.Minute)

	for range ticker.C {
		list, err := s.List()
		if err != nil {
			log.Printf("cleanup skipped: %v", err)
			continue
		}

		for _, ws := range list {
			if time.Since(ws.LastActive) > 30*time.Minute {
				if err := d.StopAndRemove(context.Background(), ws.ContainerID); err != nil {
					log.Printf("cleanup stop/remove failed: workspace=%s err=%v", ws.ID, err)
					continue
				}

				if err := s.Delete(ws.ID); err != nil {
					log.Printf("cleanup delete failed: workspace=%s err=%v", ws.ID, err)
				}
			}
		}
	}
}
