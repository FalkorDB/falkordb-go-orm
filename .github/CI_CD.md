# CI/CD Configuration

Note: this file documents CI/CD for contributors. The project overview is in the repository root `README.md`.

This directory contains GitHub Actions workflows and configuration for the FalkorDB Go ORM project.

## Workflows

### 1. Go Build & Test (`go.yml`)

**Trigger**: Push to `main` or Pull Request to `main`

**What it does**:
- Tests against Go versions: 1.21, 1.22, 1.23
- Runs with live FalkorDB instance (Docker service)
- Executes: `go build`, `go vet`, `go fmt` check
- Runs tests with race detection and coverage
- Uploads coverage to Codecov
- Caches Go modules for faster builds

**Service**: FalkorDB Docker container on port 6379

### 2. Lint (`lint.yml`)

**Trigger**: Push to `main` or Pull Request to `main`

**What it does**:
- Runs golangci-lint with multiple linters
- Configuration in `.golangci.yml`
- Checks code quality and style

**Linters enabled**:
- gofmt, govet, staticcheck
- unused, gosimple, ineffassign
- errcheck, misspell, unconvert
- unparam, gocyclo, gocritic

### 3. Release Drafter (`release-drafter.yml`)

**Trigger**: Push to `main`

**What it does**:
- Automatically generates draft release notes
- Categorizes changes by labels (features, bugs, etc.)
- Determines version based on labels (major, minor, patch)
- Configuration in `release-drafter-config.yml`

## Configuration Files

### `.golangci.yml`

Linter configuration with sensible defaults:
- Timeout: 5 minutes
- Minimum complexity: 15
- Shadow checking enabled
- Test files have relaxed rules

### `release-drafter-config.yml`

Release notes template:
- Auto-versioning based on labels
- Change categorization
- Contributor attribution

## Adding Labels to PRs

To get proper release notes, add labels to your PRs:

**Version bump**:
- `major`: Breaking changes (v1.0.0 → v2.0.0)
- `minor`: New features (v1.0.0 → v1.1.0)
- `patch`: Bug fixes (v1.0.0 → v1.0.1)

**Categories**:
- `feature`, `enhancement`: 🚀 Features section
- `fix`, `bugfix`, `bug`: 🐛 Bug Fixes section
- `chore`, `dependencies`: 🧰 Maintenance section
- `documentation`, `docs`: 📚 Documentation section

## Running Locally

### Run tests with FalkorDB

```bash
# Start FalkorDB
docker run -d -p 6379:6379 falkordb/falkordb:latest

# Run tests
go test -v -race ./...

# Stop FalkorDB
docker stop $(docker ps -q --filter ancestor=falkordb/falkordb:latest)
```

### Run linter

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

### Check formatting

```bash
# Check
gofmt -l .

# Fix
gofmt -w .
```

## CI Status

Current workflows will run on:
- Every push to `main` branch
- Every pull request to `main` branch

Check status at: https://github.com/FalkorDB/falkordb-go-orm/actions

## Based On

CI configuration follows patterns from:
- [falkordb-go](https://github.com/FalkorDB/falkordb-go) - Main Go client patterns
- FalkorDB organization standards
