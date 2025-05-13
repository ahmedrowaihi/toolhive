// Package transport provides utilities for handling different transport modes
// for communication between the client and MCP server.
package transport

import (
	"github.com/stacklok/toolhive/pkg/transport/errors"
	"github.com/stacklok/toolhive/pkg/transport/types"
)

// Factory creates transports
type Factory struct{}

// NewFactory creates a new transport factory
func NewFactory() *Factory {
	return &Factory{}
}

// Create creates a transport based on the provided configuration
func (*Factory) Create(config types.Config) (types.Transport, error) {
	switch config.Type {
	case types.TransportTypeStdio:
		return NewStdioTransport(config.Host, config.Port, config.Runtime, config.Debug, config.Middlewares...), nil
	case types.TransportTypeSSE:
		return NewSSETransport(
			config.Host,
			config.Port,
			config.TargetPort,
			config.Runtime,
			config.Debug,
			config.TargetHost,
			config.Middlewares...,
		), nil
	default:
		return nil, errors.ErrUnsupportedTransport
	}
}
