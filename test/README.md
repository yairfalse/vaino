# WGO Test Environment

Easy-to-use test environment for WGO that works on both macOS and Ubuntu.

## Quick Start

```bash
# Start the test environment
./test/test-env.sh start

# Run tests
./test/test-wgo.sh full

# Stop everything
./test/test-env.sh stop
```

## What's Included

### Kubernetes (via kind)
- Multi-tier application with:
  - Frontend (nginx, 3 replicas)
  - API server (httpbin, 2 replicas)  
  - PostgreSQL database
  - Redis cache
  - RabbitMQ message queue
- Batch jobs and CronJobs
- ConfigMaps and Secrets
- NetworkPolicies
- HorizontalPodAutoscaler
- Ingress configuration

### AWS (via LocalStack)
- S3 buckets
- EC2 instances (mocked)
- IAM resources

## Commands

### Test Environment Manager (`test-env.sh`)

```bash
./test/test-env.sh start     # Start everything
./test/test-env.sh stop      # Stop everything
./test/test-env.sh restart   # Restart
./test/test-env.sh status    # Check status
./test/test-env.sh k8s-only  # Start only Kubernetes
./test/test-env.sh aws-only  # Start only LocalStack
```

### Test Runner (`test-wgo.sh`)

```bash
./test/test-wgo.sh full   # Run full test suite
./test/test-wgo.sh scan   # Quick scan test
```

## Prerequisites

The script will check and tell you exactly what to install:

### macOS
```bash
brew install docker kind kubectl
```

### Ubuntu
```bash
# Docker
sudo apt-get update && sudo apt-get install docker.io
sudo usermod -aG docker $USER

# kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind && sudo mv ./kind /usr/local/bin/kind

# kubectl
sudo snap install kubectl --classic
```

## Test Workflow

1. **Start environment**: Creates local K8s cluster with workloads
2. **Run scan**: `wgo scan --provider kubernetes --namespace test-workloads`
3. **Create baseline**: `wgo baseline create --name test`
4. **Make changes**: Scale deployments, modify configs
5. **Detect drift**: `wgo check`
6. **View differences**: `wgo diff`

## Workload Details

The test environment includes realistic workloads:

- **Web tier**: nginx frontend with load balancer
- **API tier**: HTTP API server with environment configs
- **Data tier**: PostgreSQL with persistent storage
- **Cache tier**: Redis for session/cache
- **Messaging**: RabbitMQ for async processing
- **Jobs**: Batch processing and scheduled tasks
- **Security**: Network policies and secrets

## Tips

- The environment persists between stops/starts
- Use `status` to check what's running
- Logs: `kubectl logs -n test-workloads deployment/frontend`
- Access frontend: `kubectl port-forward -n test-workloads svc/frontend 8080:80`
- Clean rebuild: `./test/test-env.sh stop && ./test/test-env.sh start`

## Troubleshooting

### "Cannot connect to Docker"
- macOS: Make sure Docker Desktop is running
- Linux: `sudo systemctl start docker`

### "kind: command not found"
- Run the install commands shown by the script
- Make sure `/usr/local/bin` is in your PATH

### "Workloads not ready"
- Wait a bit longer: `kubectl get pods -n test-workloads --watch`
- Check events: `kubectl get events -n test-workloads`