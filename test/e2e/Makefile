# E2E Test Makefile
.PHONY: test test-aws test-drift clean setup

# Run all E2E tests
test:
	@echo "Running all E2E tests..."
	go test -v -tags=e2e ./...

# Run AWS drift detection E2E test
test-aws:
	@echo "Running AWS drift detection E2E test..."
	go test -v -tags=e2e -run TestE2EAWSDriftDetection ./...

# Run git-like diff E2E test
test-drift:
	@echo "Running git-like diff E2E test..."
	go test -v -tags=e2e -run TestE2EGitLikeDiff ./...

# Setup test environment
setup:
	@echo "Building wgo binary for E2E tests..."
	cd ../.. && go build -o test/e2e/wgo ./cmd/wgo

# Clean test artifacts
clean:
	rm -f wgo
	rm -rf tmp/