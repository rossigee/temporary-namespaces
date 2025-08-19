# Temporary namespaces

[![CI/CD](https://github.com/rossigee/temporary-namespaces/actions/workflows/build-and-publish.yaml/badge.svg)](https://github.com/rossigee/temporary-namespaces/actions/workflows/build-and-publish.yaml)

This microservice is designed to run with a role that allows it to query the annotations on all namespace, and delete any that have a specific 'expiry' timestamp annotation that has since expired.

## Requirements

- Go 1.25 or later

This allows us to spin up 'temporary namespaces' from our CI/CD build system, run a series of automated tests against them and announce them to the testing team who can perform any manual tests as required.

## Features

- **Automatic Cleanup**: Deletes namespaces based on expiry timestamp annotations
- **Safety Features**: Protects system namespaces (default, kube-system, flux-system)
- **Pattern Matching**: Configurable regex filtering for namespace names
- **Dry Run Mode**: Test cleanup operations without actual deletion
- **Prometheus Metrics**: Comprehensive monitoring and observability
- **Health Endpoints**: Built-in health checks for monitoring

## Environment Variables

- `KUBE_ANNOTATION_KEY`: Annotation key to check for expiry timestamps (required)
- `NAMESPACES_REGEX`: Regular expression to filter namespace names (optional)

## Command Line Options

- `--kubeconfig`: Path to kubeconfig file (optional, uses in-cluster config by default)
- `--dry-run`: Enable dry run mode (shows what would be deleted without actually deleting)

## Metrics and Monitoring

The service exposes Prometheus metrics on port 8080:

- **Endpoints**:
  - `/metrics` - Prometheus metrics endpoint
  - `/health` - Health check endpoint (returns "OK")

- **Available Metrics**:
  - `temporary_namespaces_checked_total{result}` - Counter of namespaces checked by result type
  - `temporary_namespaces_deleted_total{dry_run}` - Counter of namespaces deleted (by dry run status)
  - `temporary_namespaces_errors_total{type}` - Counter of errors encountered by error type
  - `temporary_namespaces_cleanup_duration_seconds` - Histogram of cleanup operation duration
  - `temporary_namespaces_current_eligible` - Gauge of namespaces currently eligible for deletion

## Development

### Building

```bash
# Build for current architecture
make build

# Build for ARM64
make build-arm64

# Build for both architectures
make all

# Clean build artifacts
make clean
```

### Testing

```bash
# Run tests
make test

# Run linter (uses golangci-lint v2.4.0)
make lint
```

### Docker

```bash
# Build container image
docker build -t temporary-namespaces .

# Build multi-platform images
docker buildx bake image-all
```

## Configuration

Start with a little [RBAC](examples/rbac.yaml).

Then, a [deployment](examples/deployment.yaml).

Put this in your flux2/ArgoCD repo :)
