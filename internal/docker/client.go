package docker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"go-cloud-ide/internal/apperr"
)

type Client struct {
	cli *client.Client
}

const workspaceReadyTimeout = 45 * time.Second

// New connects to the local Docker daemon and returns a workspace client.
func New() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, apperr.E("docker.new", apperr.KindExternal, "failed to connect to Docker", err)
	}

	return &Client{cli: cli}, nil
}

// CreateVolume creates the persistent Docker volume for a workspace.
func (c *Client) CreateVolume(ctx context.Context, name string) error {
	_, err := c.cli.VolumeCreate(ctx, volume.CreateOptions{Name: name})
	return apperr.E("docker.create_volume", apperr.KindExternal, "failed to create workspace volume", err)
}

// RunWorkspace starts a code-server container and returns its container ID and host port.
func (c *Client) RunWorkspace(ctx context.Context, name, volume string) (string, string, error) {
	port := nat.Port("80/tcp")

	resp, err := c.cli.ContainerCreate(ctx,
		&container.Config{
			Image: "islandora/code-server:4",
			Env:   []string{"PASSWORD=dev123"},
			ExposedPorts: nat.PortSet{
				"80/tcp": {},
			},
		},
		&container.HostConfig{
			Resources: container.Resources{
				Memory:   512 * 1024 * 1024,
				NanoCPUs: 1_000_000_000,
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: volume,
					Target: "/home/coder/project",
				},
			},
			PortBindings: nat.PortMap{
				port: {{HostIP: "0.0.0.0", HostPort: ""}},
			},
		},
		nil, nil, name,
	)
	if err != nil {
		return "", "", apperr.E("docker.container_create", apperr.KindExternal, "failed to create workspace container", err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", "", apperr.E("docker.container_start", apperr.KindExternal, "failed to start workspace container", err)
	}

	time.Sleep(1 * time.Second)

	inspect, err := c.cli.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return "", "", apperr.E("docker.container_inspect", apperr.KindExternal, "failed to inspect workspace container", err)
	}

	if inspect.NetworkSettings == nil {
		return "", "", apperr.E("docker.container_network_settings", apperr.KindExternal, "workspace container did not report network settings", fmt.Errorf("missing network settings"))
	}

	bindings := inspect.NetworkSettings.Ports[port]
	if len(bindings) == 0 {
		return "", "", apperr.E("docker.container_port", apperr.KindExternal, "workspace container did not expose a host port", fmt.Errorf("no port binding assigned"))
	}

	return resp.ID, bindings[0].HostPort, nil
}

// WaitUntilReady polls the workspace HTTP endpoint until it answers or the timeout is reached.
func (c *Client) WaitUntilReady(ctx context.Context, port string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, workspaceReadyTimeout)
	defer cancel()

	httpClient := &http.Client{Timeout: 1 * time.Second}
	target := "http://127.0.0.1:" + port
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return apperr.E("docker.workspace_ready.request", apperr.KindInternal, "failed to build workspace readiness request", err)
		}

		resp, err := httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			// Accept 200 (OK), 302 (Redirect), or 403 (Forbidden - nginx auth not ready yet)
			// as indicators that nginx proxy is reachable
			if resp.StatusCode == http.StatusOK ||
				resp.StatusCode == http.StatusFound ||
				resp.StatusCode == http.StatusForbidden {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return apperr.E("docker.workspace_ready.timeout", apperr.KindExternal, "workspace is still starting; try again in a moment", ctx.Err())
		case <-ticker.C:
		}
	}
}

// StopAndRemove shuts down a workspace container and deletes it from Docker.
func (c *Client) StopAndRemove(ctx context.Context, id string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	timeout := 5 * time.Second
	timeoutSeconds := int(timeout / time.Second)
	if err := c.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeoutSeconds}); err != nil {
		return apperr.E("docker.container_stop", apperr.KindExternal, "failed to stop workspace container", err)
	}

	if err := c.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true}); err != nil {
		return apperr.E("docker.container_remove", apperr.KindExternal, "failed to remove workspace container", err)
	}

	return nil
}
