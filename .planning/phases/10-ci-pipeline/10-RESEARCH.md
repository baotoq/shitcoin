# Phase 10: CI Pipeline - Research

**Researched:** 2026-03-07
**Domain:** GitHub Actions CI/CD for Go + React + Docker (GHCR)
**Confidence:** HIGH

## Summary

This phase creates GitHub Actions workflows that run Go tests, golangci-lint, frontend checks, and Docker image publishing. The project already has a solid test suite (22 test files across all domain packages, all passing), ESLint + TypeScript configured in `web/`, and multi-stage Dockerfiles for both backend and frontend from Phase 9.

The standard approach uses three workflow files: one for Go CI (test + lint), one for frontend CI (lint + typecheck + build), and one for Docker image builds. All use well-established GitHub Actions with stable APIs. golangci-lint v2 is the current major version with a new config format (`version: "2"` required in `.golangci.yml`).

**Primary recommendation:** Use `golangci/golangci-lint-action@v9` with golangci-lint v2.10, `docker/build-push-action@v6` with `docker/metadata-action@v5` for smart tagging, and standard `actions/setup-go@v6` / `actions/setup-node@v4` for language runtimes.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CI-01 | GitHub Actions runs `go test ./...` on push and PR | Standard `actions/setup-go@v6` with `go test -coverprofile` |
| CI-02 | GitHub Actions runs golangci-lint with project `.golangci.yml` config | `golangci/golangci-lint-action@v9` with v2.10, new v2 config format |
| CI-03 | GitHub Actions runs frontend lint, typecheck, and build verification | `actions/setup-node@v4` with `npm ci && npm run lint && tsc -b && npm run build` |
| CI-04 | GitHub Actions builds Docker images on PR and pushes to GHCR on main merge | `docker/build-push-action@v6` with conditional push on main, `docker/metadata-action@v5` for tags |
| CI-05 | Go test coverage is reported in CI output | `go test -coverprofile=coverage.out ./...` + `go tool cover -func=coverage.out` |
</phase_requirements>

## Standard Stack

### Core

| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| `actions/checkout` | v5 | Clone repo | Official GitHub action |
| `actions/setup-go` | v6 | Install Go, cache modules | Official, built-in caching |
| `actions/setup-node` | v4 | Install Node.js, cache npm | Official, built-in caching |
| `golangci/golangci-lint-action` | v9 | Run golangci-lint | Official action from golangci-lint authors |
| `golangci-lint` | v2.10 | Go linter aggregator | Industry standard, v2 is current major |
| `docker/setup-buildx-action` | v3 | Docker Buildx setup | Required for build-push-action |
| `docker/login-action` | v3 | GHCR authentication | Official Docker action |
| `docker/build-push-action` | v6 | Build + push images | Official, supports Buildx layer caching |
| `docker/metadata-action` | v5 | Generate image tags/labels | Smart tagging (sha, branch, latest) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| golangci-lint-action | Manual install + run | Loses smart caching and GitHub annotations |
| docker/build-push-action | Manual docker build + push | Loses Buildx cache, more verbose |
| Separate workflow files | Single monolith workflow | Monolith blocks all jobs on any failure; separate files give independent status checks |

## Architecture Patterns

### Recommended Project Structure
```
.github/
  workflows/
    ci-go.yml          # Go test + lint (CI-01, CI-02, CI-05)
    ci-frontend.yml    # Frontend lint + typecheck + build (CI-03)
    docker.yml         # Docker build + push to GHCR (CI-04)
.golangci.yml          # golangci-lint v2 config
```

### Pattern 1: Separate Workflow Files
**What:** Each concern gets its own workflow file with independent triggers
**When to use:** Always for projects with distinct build pipelines (Go + Node + Docker)
**Why:** Independent status checks on PRs, parallel execution, clear failure isolation

### Pattern 2: Conditional Docker Push
**What:** Build images on every PR (validates Dockerfile), but only push on main merge
**When to use:** CI-04 requirement -- build on PR, push on main
**Example:**
```yaml
- uses: docker/build-push-action@v6
  with:
    push: ${{ github.event_name == 'push' && github.ref == 'refs/heads/master' }}
    tags: ${{ steps.meta.outputs.tags }}
    labels: ${{ steps.meta.outputs.labels }}
```

### Pattern 3: Metadata-Driven Tagging
**What:** Use `docker/metadata-action` to auto-generate tags based on Git context
**When to use:** Always for GHCR publishing
**Example tags generated:**
- On main merge: `ghcr.io/baotoq/shitcoin:master`, `ghcr.io/baotoq/shitcoin:sha-abc1234`
- On PR: tags generated but not pushed (build-only validation)

### Anti-Patterns to Avoid
- **Single mega-workflow:** Coupling Go, frontend, and Docker in one file makes failures opaque and blocks unrelated checks
- **Hardcoded image tags:** Always use metadata-action; manual tag management drifts
- **Running lint after tests:** Lint is faster and should fail first; use separate parallel jobs or put lint first

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Go module caching | Custom cache steps | `actions/setup-go@v6` built-in cache | setup-go caches `~/go/pkg/mod` automatically |
| npm caching | Custom cache steps | `actions/setup-node@v4` with `cache: 'npm'` | Built-in, handles lockfile hashing |
| Docker layer caching | Manual save/load | `build-push-action` with `cache-from/cache-to` GHA cache | Buildx handles layer-level caching |
| Image tag generation | Shell scripts for tags | `docker/metadata-action@v5` | Handles branch, SHA, latest logic correctly |
| Lint result annotations | Parsing lint output | `golangci-lint-action` native annotations | Creates inline PR annotations automatically |

## Common Pitfalls

### Pitfall 1: GHCR Permission Denied
**What goes wrong:** Docker push fails with 403/unauthorized
**Why it happens:** `GITHUB_TOKEN` needs explicit `packages: write` permission
**How to avoid:** Add `permissions: packages: write, contents: read` to the Docker workflow
**Warning signs:** "denied: permission_denied" in push step

### Pitfall 2: golangci-lint Version Mismatch with Config
**What goes wrong:** Lint fails with config parse errors
**Why it happens:** golangci-lint v2 uses a different config format than v1
**How to avoid:** Use `version: "2"` at top of `.golangci.yml`, pin golangci-lint to v2.10 in action
**Warning signs:** "unknown configuration option" errors

### Pitfall 3: Frontend Build Missing npm ci
**What goes wrong:** Build uses stale or inconsistent dependencies
**Why it happens:** Using `npm install` instead of `npm ci` in CI
**How to avoid:** Always use `npm ci` (clean install from lockfile)
**Warning signs:** "works locally but fails in CI"

### Pitfall 4: Docker Context Path for web/Dockerfile
**What goes wrong:** Frontend Docker build fails because it can't find files
**Why it happens:** Build context must be `web/` subdirectory, not repo root
**How to avoid:** Set `context: ./web` and `file: ./web/Dockerfile` in build-push-action
**Warning signs:** "COPY failed: file not found"

### Pitfall 5: Go Test Coverage Not Displayed
**What goes wrong:** Coverage runs but percentage isn't visible in CI output
**Why it happens:** `-coverprofile` writes to file but doesn't print summary
**How to avoid:** Run `go tool cover -func=coverage.out` after tests to print per-function coverage with total
**Warning signs:** Coverage file exists but no percentage in logs

### Pitfall 6: Default Branch is master, not main
**What goes wrong:** Workflows trigger on wrong branch or Docker push doesn't fire
**Why it happens:** Project uses `master` as default branch (verified from git status)
**How to avoid:** Use `master` in all branch references, not `main`
**Warning signs:** Workflows never trigger on merge

## Code Examples

### Go CI Workflow (ci-go.yml)
```yaml
# Source: GitHub Actions official docs + golangci-lint-action README
name: Go CI

on:
  push:
    branches: [master]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
      - name: Run tests with coverage
        run: go test -coverprofile=coverage.out ./...
      - name: Display coverage
        run: go tool cover -func=coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
      - uses: golangci/golangci-lint-action@v9
        with:
          version: v2.10
```

### Frontend CI Workflow (ci-frontend.yml)
```yaml
# Source: actions/setup-node docs
name: Frontend CI

on:
  push:
    branches: [master]
  pull_request:

jobs:
  check:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: web
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
          cache-dependency-path: web/package-lock.json
      - run: npm ci
      - run: npm run lint
      - run: npx tsc -b
      - run: npm run build
```

### Docker Build + Push Workflow (docker.yml)
```yaml
# Source: docker/build-push-action README + metadata-action docs
name: Docker

on:
  push:
    branches: [master]
  pull_request:

permissions:
  contents: read
  packages: write

jobs:
  build-backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        if: github.event_name == 'push'
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ghcr.io/${{ github.repository }}
          tags: |
            type=ref,event=branch
            type=sha
      - uses: docker/build-push-action@v6
        with:
          context: .
          push: ${{ github.event_name == 'push' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  build-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        if: github.event_name == 'push'
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ghcr.io/${{ github.repository }}-web
          tags: |
            type=ref,event=branch
            type=sha
      - uses: docker/build-push-action@v6
        with:
          context: ./web
          push: ${{ github.event_name == 'push' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

### golangci-lint v2 Config (.golangci.yml)
```yaml
# Source: golangci-lint.run/docs/configuration/
version: "2"

linters:
  default: standard
  enable:
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - misspell

formatters:
  enable:
    - gofmt

issues:
  exclude-dirs:
    - web
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| golangci-lint v1 config | golangci-lint v2 with `version: "2"` | March 2025 | New config structure, `enable-all`/`disable-all` replaced by `default` |
| golangci-lint-action@v4 | golangci-lint-action@v9 (node24) | 2025 | Node.js 20 deprecated, must use v9 |
| docker/build-push-action@v5 | @v6 | 2025 | Current stable |
| Manual coverage parsing | `go tool cover -func` | Stable | Simple, no external tools needed |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + testify v1.11.1 |
| Config file | None (Go standard) |
| Quick run command | `go test ./...` |
| Full suite command | `go test -coverprofile=coverage.out ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CI-01 | Go tests run on push/PR | manual-only | Push to GitHub, verify Actions tab | N/A (workflow file) |
| CI-02 | golangci-lint runs with config | manual-only | Push to GitHub, verify lint job passes | N/A (workflow file) |
| CI-03 | Frontend lint+typecheck+build | manual-only | Push to GitHub, verify frontend job | N/A (workflow file) |
| CI-04 | Docker build on PR, push on main | manual-only | Merge to master, verify GHCR packages | N/A (workflow file) |
| CI-05 | Coverage reported in output | manual-only | Check CI logs for coverage percentage | N/A (workflow file) |

**Justification for manual-only:** CI workflow files are declarative YAML validated by GitHub Actions runtime. Local validation is limited to YAML syntax. The real test is pushing and observing the workflow runs. However, we CAN validate locally:
- `.golangci.yml` validity: `golangci-lint run` locally
- Frontend commands: `cd web && npm ci && npm run lint && npx tsc -b && npm run build`
- Go test + coverage: `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`

### Sampling Rate
- **Per task commit:** Validate YAML syntax, run local equivalents of CI commands
- **Per wave merge:** Push to GitHub and verify all workflow runs pass
- **Phase gate:** All three workflows green on a test push

### Wave 0 Gaps
- [ ] `.golangci.yml` -- golangci-lint v2 config file (does not exist yet)
- [ ] `.github/workflows/` -- directory does not exist yet
- [ ] Verify `golangci-lint run` passes locally before committing workflow

## Open Questions

1. **golangci-lint local compatibility**
   - What we know: Go 1.26.1 is used; golangci-lint v2.10 should support it
   - What's unclear: Whether all current code passes default linters without modifications
   - Recommendation: Run `golangci-lint run` locally first, fix any issues before creating the workflow

2. **GHCR image naming convention**
   - What we know: Standard is `ghcr.io/<owner>/<repo>` for backend, `ghcr.io/<owner>/<repo>-web` for frontend
   - What's unclear: Whether user has preference on naming
   - Recommendation: Use `ghcr.io/baotoq/shitcoin` (backend) and `ghcr.io/baotoq/shitcoin-web` (frontend)

## Sources

### Primary (HIGH confidence)
- [golangci/golangci-lint-action](https://github.com/golangci/golangci-lint-action) - v9 action setup, parameters, caching
- [docker/build-push-action](https://github.com/docker/build-push-action) - v6 build/push config
- [docker/metadata-action](https://github.com/docker/metadata-action) - v5 tag generation strategies
- [golangci-lint.run](https://golangci-lint.run/docs/configuration/) - v2 config format

### Secondary (MEDIUM confidence)
- [Go CI Pipeline with GitHub Actions (Dec 2025)](https://oneuptime.com/blog/post/2025-12-20-go-ci-pipeline-github-actions/view) - Workflow patterns verified with official docs
- [Docker official CI docs](https://docs.docker.com/build/ci/github-actions/) - GHCR workflow patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All tools are well-established GitHub Actions with stable APIs
- Architecture: HIGH - Three-workflow pattern is industry standard for Go+Node+Docker projects
- Pitfalls: HIGH - Based on official docs and known gotchas (GHCR permissions, v2 config format)

**Research date:** 2026-03-07
**Valid until:** 2026-04-07 (stable ecosystem, action versions pinned to major)
