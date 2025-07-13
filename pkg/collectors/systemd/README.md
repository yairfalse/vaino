# systemd Collector

The systemd collector monitors Linux system services managed by systemd, tracking their state changes, resource usage, and restart patterns.

## Features

### Core Monitoring
- **Service State Tracking**: Monitor active, failed, activating, and deactivating services
- **Real-time State Changes**: Watch service state transitions via D-Bus
- **Resource Monitoring**: Track CPU and memory usage per service
- **Dependency Tracking**: Map service dependencies and dependents

### Advanced Analysis
- **Restart Pattern Detection**: Identify flapping, degrading, or erratic restart patterns
- **Trend Analysis**: Detect increasing or decreasing restart frequencies
- **Anomaly Detection**: Identify unusual restart events and bursts
- **Predictive Analysis**: Forecast next restart based on historical patterns

### Performance
- **Low Memory Footprint**: < 20MB memory usage
- **High Throughput**: Handles 1000+ state changes per minute
- **Rate Limiting**: Configurable event rate limiting
- **Efficient Filtering**: Process only relevant services

## Requirements

- Linux operating system with systemd
- D-Bus system bus access
- Read access to systemd via D-Bus (typically requires running as root or with appropriate permissions)

## Configuration

### Basic Configuration

```yaml
systemd:
  filters:
    - "state:active,failed"
    - "type:service"
    - "exclude:user@*"
  rate_limit: 1000
  monitor_restarts: true
  monitor_resources: true
```

### Filters

Filters determine which services to monitor:

| Filter Type | Description | Example |
|------------|-------------|---------|
| `state` | Filter by service state | `state:active,failed` |
| `type` | Filter by unit type | `type:service` |
| `name` | Filter by service name | `name:nginx` |
| `exclude` | Exclude matching services | `exclude:user@*` |
| `container` | Monitor container services | `container:true` |

### Advanced Options

- `rate_limit`: Maximum events per minute (100-10000)
- `resource_poll_interval`: Resource monitoring interval in seconds
- `max_state_history`: Number of state transitions to keep per service
- `restart_analysis`: Configuration for restart pattern detection

## Usage

### Scanning Services

```bash
# Scan all systemd services
vaino scan --provider systemd

# Scan with custom configuration
vaino scan --provider systemd --config systemd.yaml

# Filter specific services
vaino scan --provider systemd --filter "name:nginx"
```

### Monitoring Changes

```bash
# Watch for service state changes
vaino watch --provider systemd --interval 30s

# Monitor only failed services
vaino watch --provider systemd --filter "state:failed"
```

### Analyzing Patterns

The collector automatically analyzes restart patterns and provides insights:

1. **Pattern Classification**:
   - `stable`: Normal operation
   - `flapping`: Rapid restart cycles
   - `degrading`: Increasing failure rate
   - `recovering`: Decreasing failure rate
   - `erratic`: Unpredictable behavior

2. **Trend Analysis**:
   - Restart frequency trends
   - Resource usage patterns
   - Dependency impact analysis

3. **Recommendations**:
   - Suggested actions based on patterns
   - Configuration optimizations
   - Troubleshooting guidance

## Resource Schema

Each systemd service is represented as a resource with the following structure:

```json
{
  "id": "systemd:service:nginx.service",
  "type": "systemd:service",
  "name": "nginx.service",
  "provider": "systemd",
  "region": "local",
  "configuration": {
    "active_state": "active",
    "sub_state": "running",
    "load_state": "loaded",
    "description": "A high performance web server",
    "main_pid": 1234,
    "restart_count": 2,
    "failure_count": 0,
    "memory_current": 52428800,
    "cpu_rate_percent": 0.5,
    "restart_pattern": {
      "pattern": "stable",
      "frequency": 0.1,
      "trend": "stable",
      "confidence": 0.9
    }
  },
  "metadata": {
    "dependencies": ["network.target", "remote-fs.target"],
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "tags": {
    "state": "active",
    "health": "healthy",
    "restart_pattern": "stable"
  }
}
```

## Integration with Container Runtimes

The systemd collector can monitor container-related services:

- Docker: `docker.service`, `containerd.service`
- Kubernetes: `kubelet.service`, `kube-proxy.service`
- Podman: `podman.service`
- CRI-O: `crio.service`

Enable container monitoring with:

```yaml
systemd:
  filters:
    - "container:true"
```

## Troubleshooting

### Permission Denied

If you encounter permission errors:

1. Run with sudo: `sudo vaino scan --provider systemd`
2. Add user to systemd-journal group: `sudo usermod -a -G systemd-journal $USER`
3. Configure PolicyKit rules for D-Bus access

### No Services Found

1. Verify systemd is running: `systemctl status`
2. Check D-Bus connection: `busctl --system`
3. Review filters in configuration

### High Memory Usage

1. Reduce `max_state_history` setting
2. Increase `resource_poll_interval`
3. Use more restrictive filters

## Performance Considerations

- **Filtering**: Use filters to reduce the number of monitored services
- **Rate Limiting**: Adjust `rate_limit` based on system activity
- **Resource Polling**: Increase `resource_poll_interval` for less frequent updates
- **History Size**: Reduce `max_state_history` to save memory

## Security Notes

- Requires read-only access to systemd via D-Bus
- No modification of service states
- No access to service logs or sensitive data
- Respects systemd security policies