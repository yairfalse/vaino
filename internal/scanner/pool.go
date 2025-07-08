package scanner

import (
	"context"
	"net/http"
	"sync"
	"time"

	"google.golang.org/api/option"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ProviderClientPool manages reusable HTTP connections and API clients across providers
type ProviderClientPool struct {
	mu         sync.RWMutex
	maxConns   int
	
	// HTTP clients for different providers
	httpClients map[string]*http.Client
	
	// AWS clients
	awsClients map[string]*AWSClientSet
	
	// GCP clients
	gcpClients map[string]*GCPClientSet
	
	// Kubernetes clients
	k8sClients map[string]*kubernetes.Clientset
	
	// Connection counters
	connCount map[string]int
	
	// Cleanup channels
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// AWSClientSet holds AWS service clients for a specific region/profile
type AWSClientSet struct {
	EC2 *ec2.Client
	S3  *s3.Client
	IAM *iam.Client
	// Add more AWS services as needed
}

// GCPClientSet holds GCP service clients for a specific project
type GCPClientSet struct {
	HTTPClient *http.Client
	Options    []option.ClientOption
	// Service clients can be created on-demand using these options
}

// NewProviderClientPool creates a new client pool with connection management
func NewProviderClientPool(maxConns int) *ProviderClientPool {
	pool := &ProviderClientPool{
		maxConns:    maxConns,
		httpClients: make(map[string]*http.Client),
		awsClients:  make(map[string]*AWSClientSet),
		gcpClients:  make(map[string]*GCPClientSet),
		k8sClients:  make(map[string]*kubernetes.Clientset),
		connCount:   make(map[string]int),
		stopCleanup: make(chan struct{}),
	}
	
	// Start cleanup goroutine
	pool.cleanupTicker = time.NewTicker(5 * time.Minute)
	go pool.cleanupLoop()
	
	return pool
}

// GetHTTPClient returns a reusable HTTP client for the given provider
func (p *ProviderClientPool) GetHTTPClient(provider string) *http.Client {
	p.mu.RLock()
	client, exists := p.httpClients[provider]
	p.mu.RUnlock()
	
	if exists {
		return client
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Double-check after acquiring write lock
	if client, exists := p.httpClients[provider]; exists {
		return client
	}
	
	// Create optimized HTTP client
	client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        p.maxConns,
			MaxIdleConnsPerHost: p.maxConns / 4,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableCompression:  false,
		},
	}
	
	p.httpClients[provider] = client
	p.connCount[provider] = 0
	
	return client
}

// GetAWSClients returns AWS service clients for the given region and profile
func (p *ProviderClientPool) GetAWSClients(ctx context.Context, region, profile string) (*AWSClientSet, error) {
	key := region + ":" + profile
	
	p.mu.RLock()
	clients, exists := p.awsClients[key]
	p.mu.RUnlock()
	
	if exists {
		return clients, nil
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Double-check after acquiring write lock
	if clients, exists := p.awsClients[key]; exists {
		return clients, nil
	}
	
	// Create AWS config with optimized settings
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return nil, err
	}
	
	// Create service clients
	clients = &AWSClientSet{
		EC2: ec2.NewFromConfig(cfg),
		S3:  s3.NewFromConfig(cfg),
		IAM: iam.NewFromConfig(cfg),
	}
	
	p.awsClients[key] = clients
	p.connCount[key] = 0
	
	return clients, nil
}

// GetGCPClients returns GCP client configuration for the given project
func (p *ProviderClientPool) GetGCPClients(ctx context.Context, projectID, credentialsFile string) (*GCPClientSet, error) {
	key := projectID + ":" + credentialsFile
	
	p.mu.RLock()
	clients, exists := p.gcpClients[key]
	p.mu.RUnlock()
	
	if exists {
		return clients, nil
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Double-check after acquiring write lock
	if clients, exists := p.gcpClients[key]; exists {
		return clients, nil
	}
	
	// Create HTTP client with connection pooling
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        p.maxConns,
			MaxIdleConnsPerHost: p.maxConns / 4,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	
	// Prepare client options
	var options []option.ClientOption
	options = append(options, option.WithHTTPClient(httpClient))
	
	if credentialsFile != "" {
		options = append(options, option.WithCredentialsFile(credentialsFile))
	}
	
	clients = &GCPClientSet{
		HTTPClient: httpClient,
		Options:    options,
	}
	
	p.gcpClients[key] = clients
	p.connCount[key] = 0
	
	return clients, nil
}

// GetKubernetesClient returns a Kubernetes client for the given context
func (p *ProviderClientPool) GetKubernetesClient(context string) (*kubernetes.Clientset, error) {
	p.mu.RLock()
	client, exists := p.k8sClients[context]
	p.mu.RUnlock()
	
	if exists {
		return client, nil
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Double-check after acquiring write lock
	if client, exists := p.k8sClients[context]; exists {
		return client, nil
	}
	
	// Create Kubernetes config
	var config *rest.Config
	var err error
	
	if context == "" {
		// Use in-cluster config if available
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig
			config, err = clientcmd.BuildConfigFromFlags("", "")
		}
	} else {
		// Use specific context
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{CurrentContext: context},
		).ClientConfig()
	}
	
	if err != nil {
		return nil, err
	}
	
	// Create clientset
	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	
	p.k8sClients[context] = client
	p.connCount[context] = 0
	
	return client, nil
}

// IncrementConnCount increments the connection count for tracking
func (p *ProviderClientPool) IncrementConnCount(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connCount[key]++
}

// GetConnCount returns the current connection count for a key
func (p *ProviderClientPool) GetConnCount(key string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connCount[key]
}

// cleanupLoop periodically cleans up unused connections
func (p *ProviderClientPool) cleanupLoop() {
	for {
		select {
		case <-p.cleanupTicker.C:
			p.cleanupUnusedConnections()
		case <-p.stopCleanup:
			return
		}
	}
}

// cleanupUnusedConnections removes connections that haven't been used recently
func (p *ProviderClientPool) cleanupUnusedConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Simple cleanup based on usage count
	// In a production system, you'd want to track last access time
	for key, count := range p.connCount {
		if count == 0 {
			// Remove unused connections
			delete(p.httpClients, key)
			delete(p.awsClients, key)
			delete(p.gcpClients, key)
			delete(p.k8sClients, key)
			delete(p.connCount, key)
		} else {
			// Reset counters
			p.connCount[key] = 0
		}
	}
}

// GetStats returns statistics about the connection pool
func (p *ProviderClientPool) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return map[string]interface{}{
		"max_connections":     p.maxConns,
		"http_clients":        len(p.httpClients),
		"aws_clients":         len(p.awsClients),
		"gcp_clients":         len(p.gcpClients),
		"kubernetes_clients":  len(p.k8sClients),
		"total_connections":   len(p.connCount),
	}
}

// Close shuts down the connection pool and cleans up resources
func (p *ProviderClientPool) Close() error {
	close(p.stopCleanup)
	p.cleanupTicker.Stop()
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Close all HTTP clients
	for _, client := range p.httpClients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	
	// Clear all maps
	p.httpClients = make(map[string]*http.Client)
	p.awsClients = make(map[string]*AWSClientSet)
	p.gcpClients = make(map[string]*GCPClientSet)
	p.k8sClients = make(map[string]*kubernetes.Clientset)
	p.connCount = make(map[string]int)
	
	return nil
}