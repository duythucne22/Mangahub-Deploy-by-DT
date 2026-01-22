package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all TUI configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Protocol ProtocolConfig `yaml:"protocol"`
	UI       UIConfig       `yaml:"ui"`
}

// ServerConfig contains server connection settings
type ServerConfig struct {
	Host string `yaml:"host"`
	HTTP HTTPConfig `yaml:"http"`
	GRPC GRPCConfig `yaml:"grpc"`
	WS   WSConfig   `yaml:"ws"`
	UDP  UDPConfig  `yaml:"udp"`
	TCP  TCPConfig  `yaml:"tcp"`
}

// HTTPConfig for REST API
type HTTPConfig struct {
	Port    int    `yaml:"port"`
	BaseURL string `yaml:"base_url"`
}

// GRPCConfig for gRPC search service
type GRPCConfig struct {
	Port int    `yaml:"port"`
	Addr string `yaml:"addr"`
}

// WSConfig for WebSocket chat
type WSConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
	URL  string `yaml:"url"`
}

// UDPConfig for UDP notifications
type UDPConfig struct {
	Port int    `yaml:"port"`
	Addr string `yaml:"addr"`
}

// TCPConfig for TCP statistics
type TCPConfig struct {
	Port int    `yaml:"port"`
	Addr string `yaml:"addr"`
}

// ProtocolConfig defines protocol preferences
type ProtocolConfig struct {
	PreferredSearch string `yaml:"preferred_search"` // "http" or "grpc"
	EnableUDP       bool   `yaml:"enable_udp"`
	EnableTCP       bool   `yaml:"enable_tcp"`
	EnableWebSocket bool   `yaml:"enable_websocket"`
}

// UIConfig for UI preferences
type UIConfig struct {
	Theme          string `yaml:"theme"`
	RefreshRate    int    `yaml:"refresh_rate_ms"`
	PageSize       int    `yaml:"page_size"`
	EnableAnimations bool `yaml:"enable_animations"`
}

// Default returns default configuration
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			HTTP: HTTPConfig{
				Port:    8080,
				BaseURL: "http://localhost:8080/api/v1",
			},
			GRPC: GRPCConfig{
				Port: 50051,
				Addr: "localhost:50051",
			},
			WS: WSConfig{
				Port: 8080,
				Path: "/ws/manga",
				URL:  "ws://localhost:8080/ws/manga",
			},
			UDP: UDPConfig{
				Port: 4000,
				Addr: "localhost:4000",
			},
			TCP: TCPConfig{
				Port: 6000,
				Addr: "localhost:6000",
			},
		},
		Protocol: ProtocolConfig{
			PreferredSearch: "grpc",
			EnableUDP:       true,
			EnableTCP:       true,
			EnableWebSocket: true,
		},
		UI: UIConfig{
			Theme:            "dracula",
			RefreshRate:      1000,
			PageSize:         20,
			EnableAnimations: true,
		},
	}
}

// Load loads configuration from file, falling back to defaults
func Load(configPath string) (*Config, error) {
	// If no config path provided, use default locations
	if configPath == "" {
		configPath = findConfigFile()
	}

	// If still no config found, use defaults
	if configPath == "" {
		return Default(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Default(), nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Fill in computed fields
	// Detect if host is a public domain (uses HTTPS/WSS) vs localhost (HTTP/WS)
	httpScheme := "http"
	wsScheme := "ws"
	if cfg.Server.Host != "localhost" && cfg.Server.Host != "127.0.0.1" {
		httpScheme = "https"
		wsScheme = "wss"
	}
	
	cfg.Server.HTTP.BaseURL = fmt.Sprintf("%s://%s:%d/api/v1", 
		httpScheme, cfg.Server.Host, cfg.Server.HTTP.Port)
	cfg.Server.GRPC.Addr = fmt.Sprintf("%s:%d", 
		cfg.Server.Host, cfg.Server.GRPC.Port)
	cfg.Server.WS.URL = fmt.Sprintf("%s://%s:%d%s", 
		wsScheme, cfg.Server.Host, cfg.Server.WS.Port, cfg.Server.WS.Path)
	cfg.Server.UDP.Addr = fmt.Sprintf("%s:%d", 
		cfg.Server.Host, cfg.Server.UDP.Port)
	cfg.Server.TCP.Addr = fmt.Sprintf("%s:%d", 
		cfg.Server.Host, cfg.Server.TCP.Port)

	return &cfg, nil
}

// Save saves configuration to file
func (c *Config) Save(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// findConfigFile searches for config in standard locations
func findConfigFile() string {
	locations := []string{
		"./mangahub-tui.yaml",
		"./config/tui.yaml",
		filepath.Join(os.Getenv("HOME"), ".config", "mangahub", "tui.yaml"),
		filepath.Join(os.Getenv("HOME"), ".mangahub-tui.yaml"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}

// GetHTTPBaseURL returns the computed HTTP base URL
func (c *Config) GetHTTPBaseURL() string {
	if c.Server.HTTP.BaseURL != "" {
		return c.Server.HTTP.BaseURL
	}
	return fmt.Sprintf("http://%s:%d/api/v1", c.Server.Host, c.Server.HTTP.Port)
}

// GetGRPCAddr returns the computed gRPC address
func (c *Config) GetGRPCAddr() string {
	if c.Server.GRPC.Addr != "" {
		return c.Server.GRPC.Addr
	}
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.GRPC.Port)
}

// GetWebSocketURL returns the computed WebSocket URL
func (c *Config) GetWebSocketURL() string {
	if c.Server.WS.URL != "" {
		return c.Server.WS.URL
	}
	return fmt.Sprintf("ws://%s:%d%s", c.Server.Host, c.Server.WS.Port, c.Server.WS.Path)
}
