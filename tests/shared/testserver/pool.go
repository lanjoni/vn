package testserver

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

type ServerPool interface {
	GetServer(config ServerConfig) (*TestServer, error)
	ReleaseServer(server *TestServer)
	Shutdown()
}

type ServerConfig struct {
	Handler    http.Handler
	Port       int
	TLS        bool
	Timeout    time.Duration
	ConfigName string
}

type TestServer struct {
	URL     string
	Port    int
	Handler http.Handler
	server  *httptest.Server
	config  ServerConfig
	inUse   bool
}

type serverPool struct {
	mu      sync.Mutex
	servers map[string]*TestServer
	ports   map[int]bool
}

func NewServerPool() ServerPool {
	return &serverPool{
		servers: make(map[string]*TestServer),
		ports:   make(map[int]bool),
	}
}

func (sp *serverPool) GetServer(config ServerConfig) (*TestServer, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	configKey := sp.getConfigKey(config)

	if server, exists := sp.servers[configKey]; exists && !server.inUse {
		server.inUse = true
		return server, nil
	}

	server, err := sp.createServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	server.inUse = true
	sp.servers[configKey] = server

	return server, nil
}

func (sp *serverPool) ReleaseServer(server *TestServer) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	server.inUse = false
}

func (sp *serverPool) createServer(config ServerConfig) (*TestServer, error) {
	var server *httptest.Server

	if config.TLS {
		server = httptest.NewTLSServer(config.Handler)
	} else if config.Port != 0 {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
		if err != nil {
			return nil, fmt.Errorf("failed to listen on port %d: %w", config.Port, err)
		}
		server = httptest.NewUnstartedServer(config.Handler)
		server.Listener = listener
		server.Start()
		sp.ports[config.Port] = true
	} else {
		server = httptest.NewServer(config.Handler)
	}

	port := sp.extractPort(server.URL)

	testServer := &TestServer{
		URL:     server.URL,
		Port:    port,
		Handler: config.Handler,
		server:  server,
		config:  config,
	}

	return testServer, nil
}

func (sp *serverPool) extractPort(url string) int {
	_, portStr, err := net.SplitHostPort(url[7:]) // Remove "http://"
	if err != nil {
		return 0
	}

	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port) //nolint:errcheck
	return port
}

func (sp *serverPool) getConfigKey(config ServerConfig) string {
	if config.ConfigName != "" {
		return config.ConfigName
	}

	return fmt.Sprintf("port_%d_tls_%t", config.Port, config.TLS)
}

func (sp *serverPool) Shutdown() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	for _, server := range sp.servers {
		if server.server != nil {
			server.server.Close()
		}
	}

	sp.servers = make(map[string]*TestServer)
	sp.ports = make(map[int]bool)
}

func (ts *TestServer) Close() {
	if ts.server != nil {
		ts.server.Close()
	}
}

func (ts *TestServer) Client() *http.Client {
	if ts.server != nil {
		return ts.server.Client()
	}
	return http.DefaultClient
}
