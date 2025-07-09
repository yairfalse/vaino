# WGO Architecture Overview

This document provides a comprehensive overview of WGO's architecture, including system design, command structure, and performance optimizations.

## System Architecture

### Core Components

WGO follows a modular architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                    WGO Core System                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │  Commands   │  │  Scanner    │  │  Analyzer   │        │
│  │  Layer      │  │  Engine     │  │  Engine     │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
│         │                 │                 │             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ Collectors  │  │   Storage   │  │ Correlation │        │
│  │  (Providers)│  │   Layer     │  │   Engine    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
│         │                 │                 │             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │            Infrastructure Providers                    │ │
│  │  AWS  │  GCP  │  Kubernetes  │  Terraform  │  Others  │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

#### 1. Commands Layer
- **CLI Interface**: Cobra-based command structure
- **Input Validation**: Parameter validation and sanitization
- **Output Formatting**: Multiple output formats (table, JSON, markdown)
- **Configuration Management**: Configuration file and environment variable handling

#### 2. Scanner Engine
- **Provider Orchestration**: Manages multiple infrastructure providers
- **Concurrent Scanning**: Parallel resource collection for performance
- **Resource Normalization**: Standardizes resource representations
- **State Management**: Snapshot creation and storage

#### 3. Analyzer Engine
- **Drift Detection**: Compares infrastructure states
- **Change Classification**: Categorizes changes by type and severity
- **AI Analysis**: Anthropic Claude integration for intelligent insights
- **Report Generation**: Generates comprehensive drift reports

#### 4. Collectors (Providers)
- **AWS Collector**: EC2, S3, Lambda, RDS, IAM resources
- **GCP Collector**: Compute Engine, Cloud Storage, GKE resources
- **Kubernetes Collector**: Pods, Services, Deployments, ConfigMaps
- **Terraform Collector**: State file parsing and resource extraction

#### 5. Storage Layer
- **Snapshot Storage**: Persistent storage for infrastructure states
- **Metadata Management**: Timestamps, tags, and state relationships
- **Indexing**: Fast retrieval and querying of historical data
- **Data Integrity**: Checksums and validation

#### 6. Correlation Engine
- **Pattern Detection**: Identifies related infrastructure changes
- **Timeline Analysis**: Temporal correlation of changes
- **Confidence Scoring**: Quantifies correlation strength
- **Parallel Processing**: Concurrent correlation for large datasets

## Command Architecture

### Current Command Structure

```bash
wgo scan         # Infrastructure scanning
wgo baseline     # Baseline management (deprecated)
wgo diff         # Drift detection
wgo check        # Status checking
wgo explain      # AI-powered analysis
wgo auth         # Authentication management
wgo version      # Version information
```

### Proposed Command Structure (Design Phase)

Based on user feedback and usability studies, the command structure is being redesigned:

```bash
# Primary Commands
wgo scan         # Scan infrastructure and detect drift
wgo drift        # Show drift details (replaces diff/check)
wgo snapshots    # Manage saved infrastructure states

# Secondary Commands
wgo auth         # Authentication management
wgo explain      # AI-powered drift analysis
wgo version      # Version information
```

#### Migration Strategy

**Phase 1: Foundation (No Breaking Changes)**
- Implement new `drift` command alongside existing commands
- Add `snapshots` command for state management
- Internal refactoring without changing public APIs

**Phase 2: Deprecation (Backward Compatible)**
- Add deprecation warnings to old commands
- Provide migration guidance
- Update documentation to show new patterns

**Phase 3: Cleanup (Major Version)**
- Remove deprecated commands
- Simplify command structure
- Complete migration to new user experience

## Performance Architecture

### Concurrent Scanning System

WGO implements a sophisticated concurrent scanning system for optimal performance:

```
┌─────────────────────────────────────────────────────────────┐
│                 ConcurrentScanner                           │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────┐ │
│ │    AWS      │ │    GCP      │ │ Kubernetes  │ │Terraform│ │
│ │ Collector   │ │ Collector   │ │ Collector   │ │Collector│ │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────┘ │
│         │               │               │             │     │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────┐ │
│ │  Worker     │ │  Worker     │ │  Worker     │ │ Worker  │ │
│ │  Pool       │ │  Pool       │ │  Pool       │ │ Pool    │ │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────┘ │
│         │               │               │             │     │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │            Connection Pool Manager                     │ │
│ └─────────────────────────────────────────────────────────┘ │
│         │                                                   │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │          Resource Merger & Deduplicator                │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Performance Characteristics

| Providers | Sequential | Concurrent | Speedup |
|-----------|------------|------------|---------|
| 2         | 30s        | 12s        | 2.5x    |
| 3         | 45s        | 15s        | 3.0x    |
| 4         | 60s        | 18s        | 3.3x    |
| 5+        | 90s        | 22s        | 4.1x    |

### Optimization Strategies

1. **Worker Pool Management**
   - Controlled concurrent operations
   - Dynamic worker scaling based on load
   - Resource-aware worker allocation

2. **Connection Pooling**
   - HTTP connection reuse
   - Connection multiplexing
   - Timeout and retry management

3. **Memory Optimization**
   - Streaming resource processing
   - Batched operations
   - Efficient data structures

4. **API Call Optimization**
   - Parallel API calls within providers
   - Request batching where possible
   - Intelligent rate limiting

## Data Architecture

### State Storage Format

```go
type StateMetadata struct {
    ID          string              `json:"id"`
    Name        string              `json:"name,omitempty"`
    Timestamp   time.Time           `json:"timestamp"`
    Provider    string              `json:"provider"`
    Tags        map[string]string   `json:"tags,omitempty"`
    
    // Enhanced metadata
    AutoName    string              `json:"auto_name,omitempty"`
    GitCommit   string              `json:"git_commit,omitempty"`
    Environment string              `json:"environment,omitempty"`
    
    // Performance metrics
    ScanDuration time.Duration      `json:"scan_duration"`
    ResourceCount int               `json:"resource_count"`
}

type Snapshot struct {
    Metadata  StateMetadata         `json:"metadata"`
    Resources []Resource            `json:"resources"`
    Checksum  string                `json:"checksum"`
}
```

### Resource Representation

```go
type Resource struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"`
    Provider    string                 `json:"provider"`
    Region      string                 `json:"region,omitempty"`
    Name        string                 `json:"name,omitempty"`
    Properties  map[string]interface{} `json:"properties"`
    Tags        map[string]string      `json:"tags,omitempty"`
    
    // Metadata
    CreatedAt   *time.Time            `json:"created_at,omitempty"`
    UpdatedAt   *time.Time            `json:"updated_at,omitempty"`
    
    // Relationships
    Dependencies []string              `json:"dependencies,omitempty"`
    Children     []string              `json:"children,omitempty"`
}
```

## Correlation Engine Architecture

### Concurrent Correlation System

The correlation engine uses parallel processing to identify related infrastructure changes:

```go
type ConcurrentCorrelator struct {
    timeWindow     time.Duration
    workerCount    int
    patternWorkers []PatternWorker
    mutex          sync.RWMutex
}

type PatternWorker struct {
    matcher   PatternMatcher
    input     chan []Change
    output    chan CorrelationResult
    closeOnce sync.Once
}
```

### Pattern Matching Types

1. **Scaling Pattern Matcher**
   - Detects resource scaling events
   - Identifies capacity changes
   - Correlates related scaling actions

2. **Configuration Update Matcher**
   - Tracks configuration changes
   - Identifies update cascades
   - Correlates related config updates

3. **Service Deployment Matcher**
   - Detects deployment patterns
   - Identifies service rollouts
   - Correlates deployment-related changes

4. **Network Pattern Matcher**
   - Tracks network configuration changes
   - Identifies connectivity modifications
   - Correlates network-related events

5. **Storage Pattern Matcher**
   - Detects storage changes
   - Identifies data migration patterns
   - Correlates storage-related modifications

6. **Security Pattern Matcher**
   - Tracks security policy changes
   - Identifies access modifications
   - Correlates security-related events

### Performance Optimization

- **Worker Pool Scaling**: 2-8 workers based on CPU cores
- **Channel-based Communication**: Efficient inter-worker communication
- **Batched Processing**: Processes changes in batches for efficiency
- **Race Condition Prevention**: Proper synchronization with sync.Once

## Authentication Architecture

### Provider Authentication

WGO supports multiple authentication methods for different providers:

#### AWS Authentication
- **IAM Roles**: Preferred for production environments
- **Access Keys**: For development and testing
- **Instance Profiles**: For EC2-based deployments
- **Assume Role**: For cross-account access

#### GCP Authentication
- **Service Account Keys**: JSON key files
- **Application Default Credentials**: For GCP environments
- **User Credentials**: For development
- **Workload Identity**: For Kubernetes deployments

#### Kubernetes Authentication
- **Kubeconfig Files**: Standard configuration
- **Service Account Tokens**: For in-cluster access
- **OIDC**: For enterprise environments
- **Client Certificates**: For mutual TLS

### Security Best Practices

1. **Credential Management**
   - Environment variable preference
   - Secure credential storage
   - Regular credential rotation

2. **Access Control**
   - Least privilege principle
   - Role-based access control
   - Audit logging

3. **Network Security**
   - TLS/SSL enforcement
   - Certificate validation
   - Secure connection pooling

## Error Handling Architecture

### Error Classification

```go
type ErrorType string

const (
    ErrorTypeAuthentication ErrorType = "authentication"
    ErrorTypeAuthorization  ErrorType = "authorization"
    ErrorTypeNetwork        ErrorType = "network"
    ErrorTypeRateLimit      ErrorType = "rate_limit"
    ErrorTypeProvider       ErrorType = "provider"
    ErrorTypeConfiguration  ErrorType = "configuration"
    ErrorTypeInternal       ErrorType = "internal"
)

type WGOError struct {
    Type        ErrorType         `json:"type"`
    Message     string            `json:"message"`
    Provider    string            `json:"provider,omitempty"`
    Resource    string            `json:"resource,omitempty"`
    Suggestions []string          `json:"suggestions,omitempty"`
    RetryAfter  *time.Duration    `json:"retry_after,omitempty"`
}
```

### Error Recovery Strategies

1. **Graceful Degradation**
   - Continue operation with partial results
   - Provide meaningful error messages
   - Suggest corrective actions

2. **Retry Logic**
   - Exponential backoff for transient failures
   - Configurable retry attempts
   - Circuit breaker pattern

3. **User Guidance**
   - Clear error messages
   - Actionable suggestions
   - Links to documentation

## Configuration Architecture

### Configuration Hierarchy

1. **Command Line Arguments** (Highest Priority)
2. **Environment Variables**
3. **Configuration Files**
4. **Default Values** (Lowest Priority)

### Configuration File Format

```yaml
# ~/.wgo/config.yaml
providers:
  aws:
    profile: default
    regions: ["us-east-1", "us-west-2"]
    timeout: 5m
    
  gcp:
    project_id: "my-project-123"
    regions: ["us-central1", "us-east1"]
    credentials: "/path/to/service-account.json"
    
  kubernetes:
    kubeconfig: "~/.kube/config"
    namespaces: ["default", "kube-system"]

scanning:
  concurrent: true
  max_workers: 4
  timeout: 10m
  
storage:
  base_path: "~/.wgo"
  compression: true
  retention_days: 90

output:
  format: "table"
  color: true
  quiet: false
```

## Future Architecture Considerations

### Planned Enhancements

1. **Distributed Scanning**
   - Multi-node scanning support
   - Load balancing across nodes
   - Centralized result aggregation

2. **Real-time Monitoring**
   - Streaming change detection
   - WebSocket-based updates
   - Event-driven architecture

3. **Machine Learning Integration**
   - Anomaly detection
   - Predictive drift analysis
   - Automated remediation suggestions

4. **API Gateway**
   - RESTful API exposure
   - GraphQL support
   - Webhook integrations

5. **Plugin System**
   - Custom provider plugins
   - Extension points
   - Third-party integrations

### Scalability Considerations

1. **Horizontal Scaling**
   - Container orchestration support
   - Stateless service design
   - Distributed caching

2. **Performance Optimization**
   - Advanced caching strategies
   - Database optimization
   - Memory management improvements

3. **Monitoring and Observability**
   - Metrics collection
   - Distributed tracing
   - Performance profiling

This architecture provides a solid foundation for WGO's continued growth and evolution, ensuring scalability, maintainability, and extensibility for future enhancements.