// Package main demonstrates how to create a FlexCore plugin using HashiCorp go-plugin
package main

import (
	"context"
	"log"
	"net/rpc"
	"strings"
	"time"

	"github.com/flext/flexcore/pkg/adapter"
	"github.com/hashicorp/go-plugin"
)

// SimpleAdapter is a basic adapter implementation
type SimpleAdapter struct {
	config map[string]interface{}
}

func (a *SimpleAdapter) Name() string {
	return "simple-adapter"
}

func (a *SimpleAdapter) Version() string {
	return "1.0.0"
}

func (a *SimpleAdapter) Configure(config map[string]interface{}) error {
	a.config = config
	log.Printf("Configured with: %+v", config)
	return nil
}

func (a *SimpleAdapter) Extract(ctx context.Context, req adapter.ExtractRequest) (*adapter.ExtractResponse, error) {
	log.Printf("Extract called with source: %s", req.Source)
	
	// Simulate some data extraction
	records := []adapter.Record{
		{
			ID: "1",
			Data: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   30,
			},
			Timestamp: time.Now(),
		},
		{
			ID: "2",
			Data: map[string]interface{}{
				"name":  "Jane Smith",
				"email": "jane@example.com",
				"age":   25,
			},
			Timestamp: time.Now(),
		},
	}

	return &adapter.ExtractResponse{
		Records:     records,
		HasMore:     false,
		ExtractedAt: time.Now(),
	}, nil
}

func (a *SimpleAdapter) Load(ctx context.Context, req adapter.LoadRequest) (*adapter.LoadResponse, error) {
	log.Printf("Load called with target: %s, records: %d", req.Target, len(req.Records))
	
	// Simulate loading data
	return &adapter.LoadResponse{
		RecordsLoaded: int64(len(req.Records)),
		RecordsFailed: 0,
		LoadedAt:      time.Now(),
	}, nil
}

func (a *SimpleAdapter) Transform(ctx context.Context, req adapter.TransformRequest) (*adapter.TransformResponse, error) {
	log.Printf("Transform called with %d records", len(req.Records))
	
	// Simple transformation: uppercase all string values
	transformed := make([]adapter.Record, len(req.Records))
	
	for i, record := range req.Records {
		newData := make(map[string]interface{})
		for k, v := range record.Data {
			if str, ok := v.(string); ok {
				newData[k] = strings.ToUpper(str)
			} else {
				newData[k] = v
			}
		}
		
		transformed[i] = adapter.Record{
			ID:        record.ID,
			Data:      newData,
			Timestamp: time.Now(),
		}
	}
	
	return &adapter.TransformResponse{
		Records:            transformed,
		RecordsTransformed: int64(len(transformed)),
		RecordsFailed:      0,
		TransformedAt:      time.Now(),
	}, nil
}

func (a *SimpleAdapter) HealthCheck(ctx context.Context) error {
	// Always healthy in this simple example
	return nil
}

// AdapterPlugin is the implementation of plugin.Plugin for our adapter
type AdapterPlugin struct {
	Impl adapter.Adapter
}

func (p *AdapterPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &AdapterRPCServer{Impl: p.Impl}, nil
}

func (p *AdapterPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &AdapterRPCClient{client: c}, nil
}

// AdapterRPCServer is the RPC server that AdapterRPC talks to
type AdapterRPCServer struct {
	Impl adapter.Adapter
}

func (s *AdapterRPCServer) Name(args interface{}, resp *string) error {
	*resp = s.Impl.Name()
	return nil
}

func (s *AdapterRPCServer) Version(args interface{}, resp *string) error {
	*resp = s.Impl.Version()
	return nil
}

func (s *AdapterRPCServer) Configure(args map[string]interface{}, resp *interface{}) error {
	return s.Impl.Configure(args)
}

func (s *AdapterRPCServer) Extract(args adapter.ExtractRequest, resp *adapter.ExtractResponse) error {
	result, err := s.Impl.Extract(context.Background(), args)
	if err != nil {
		return err
	}
	*resp = *result
	return nil
}

func (s *AdapterRPCServer) Load(args adapter.LoadRequest, resp *adapter.LoadResponse) error {
	result, err := s.Impl.Load(context.Background(), args)
	if err != nil {
		return err
	}
	*resp = *result
	return nil
}

func (s *AdapterRPCServer) Transform(args adapter.TransformRequest, resp *adapter.TransformResponse) error {
	result, err := s.Impl.Transform(context.Background(), args)
	if err != nil {
		return err
	}
	*resp = *result
	return nil
}

func (s *AdapterRPCServer) HealthCheck(args interface{}, resp *interface{}) error {
	return s.Impl.HealthCheck(context.Background())
}

// AdapterRPCClient is an implementation that talks over RPC
type AdapterRPCClient struct {
	client *rpc.Client
}

func (c *AdapterRPCClient) Name() string {
	var resp string
	c.client.Call("Plugin.Name", new(interface{}), &resp)
	return resp
}

func (c *AdapterRPCClient) Version() string {
	var resp string
	c.client.Call("Plugin.Version", new(interface{}), &resp)
	return resp
}

func (c *AdapterRPCClient) Configure(config map[string]interface{}) error {
	var resp interface{}
	return c.client.Call("Plugin.Configure", config, &resp)
}

func (c *AdapterRPCClient) Extract(ctx context.Context, req adapter.ExtractRequest) (*adapter.ExtractResponse, error) {
	var resp adapter.ExtractResponse
	err := c.client.Call("Plugin.Extract", req, &resp)
	return &resp, err
}

func (c *AdapterRPCClient) Load(ctx context.Context, req adapter.LoadRequest) (*adapter.LoadResponse, error) {
	var resp adapter.LoadResponse
	err := c.client.Call("Plugin.Load", req, &resp)
	return &resp, err
}

func (c *AdapterRPCClient) Transform(ctx context.Context, req adapter.TransformRequest) (*adapter.TransformResponse, error) {
	var resp adapter.TransformResponse
	err := c.client.Call("Plugin.Transform", req, &resp)
	return &resp, err
}

func (c *AdapterRPCClient) HealthCheck(ctx context.Context) error {
	var resp interface{}
	return c.client.Call("Plugin.HealthCheck", new(interface{}), &resp)
}

// Handshake is a common handshake that is shared by plugin and host
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "FLEXCORE_PLUGIN",
	MagicCookieValue: "flexcore-adapter-v1",
}

// pluginMap is the map of plugins we can dispense
var pluginMap = map[string]plugin.Plugin{
	"adapter": &AdapterPlugin{},
}

func main() {
	// Create adapter instance
	adapter := &SimpleAdapter{}

	// Serve the plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"adapter": &AdapterPlugin{Impl: adapter},
		},
		
		// A non-nil value here enables gRPC serving for this plugin
		GRPCServer: plugin.DefaultGRPCServer,
		
		// If set, the plugin will exit gracefully after this timeout
		// if it doesn't receive any connections
		// Timeout: 30 * time.Second, // Removed - not supported in current go-plugin version
	})
}