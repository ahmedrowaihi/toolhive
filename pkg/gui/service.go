package gui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/stacklok/toolhive/pkg/client"
	"github.com/stacklok/toolhive/pkg/labels"
	"github.com/stacklok/toolhive/pkg/lifecycle"
	"github.com/stacklok/toolhive/pkg/registry"
	"github.com/stacklok/toolhive/pkg/runner"
)

type Service struct {
	token string
}

func NewService() *Service {
	return &Service{
		token: os.Getenv("TOOLHIVE_AUTH_TOKEN"),
	}
}

func (s *Service) GetToken() string {
	return s.token
}

func (s *Service) ListServers(ctx context.Context) ([]ServerInfo, error) {

	manager, err := lifecycle.NewManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create container manager: %v", err)
	}

	containers, err := manager.ListContainers(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %v", err)
	}

	var servers []ServerInfo
	for _, c := range containers {

		name := labels.GetContainerName(c.Labels)
		if name == "" {
			name = c.Name
		}

		transport := labels.GetTransportType(c.Labels)
		if transport == "" {
			transport = "unknown"
		}

		toolType := labels.GetToolType(c.Labels)

		port, err := labels.GetPort(c.Labels)
		if err != nil {
			port = 0
		}

		url := ""
		if port > 0 {
			url = client.GenerateMCPServerURL("localhost", port, name)
		}

		servers = append(servers, ServerInfo{
			ID:        c.ID,
			Name:      name,
			Image:     c.Image,
			State:     c.State,
			Transport: transport,
			ToolType:  toolType,
			Port:      port,
			URL:       url,
		})
	}

	return servers, nil
}

func (s *Service) RunServer(ctx context.Context, name string) (string, error) {
	config := &runner.RunConfig{
		Name: name,
	}

	cmd := exec.CommandContext(ctx, "thv", "run", config.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run server: %w", err)
	}
	return string(output), nil
}

func (s *Service) StopServer(ctx context.Context, name string) error {

	manager, err := lifecycle.NewManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create container manager: %v", err)
	}

	if err := manager.DeleteContainer(ctx, name, true); err != nil {
		return fmt.Errorf("failed to force stop container: %v", err)
	}

	return nil
}

func (s *Service) RestartServer(ctx context.Context, name string) error {

	manager, err := lifecycle.NewManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create container manager: %v", err)
	}

	if err := manager.RestartContainer(ctx, name); err != nil {
		return fmt.Errorf("failed to restart server: %v", err)
	}

	return nil
}

func (s *Service) SearchRegistry(query string) ([]*registry.Server, error) {
	return registry.SearchServers(query)
}

func (s *Service) RunCommand(ctx context.Context, command string) (string, error) {

	if strings.HasPrefix(strings.ToLower(command), "thv ") {
		command = command[4:]
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid command")
	}

	cmd := exec.CommandContext(ctx, "thv", parts...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
