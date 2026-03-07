---
phase: 10
slug: ci-pipeline
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-07
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + ESLint + TypeScript compiler |
| **Config file** | .golangci.yml (new), web/eslint.config.js (existing) |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -coverprofile=coverage.out ./... && cd web && npm ci && npm run lint && npx tsc -b && npm run build` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run local equivalents of CI commands
- **After every plan wave:** Validate YAML syntax + run local commands
- **Before `/gsd:verify-work`:** Push to GitHub, verify all workflows green
- **Max feedback latency:** 30 seconds (local), requires push for full CI validation

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | CI-01, CI-02, CI-05 | local | `go test -coverprofile=coverage.out ./... && golangci-lint run` | N/A W0 | pending |
| 10-02-01 | 02 | 1 | CI-03 | local | `cd web && npm ci && npm run lint && npx tsc -b && npm run build` | N/A W0 | pending |
| 10-03-01 | 03 | 1 | CI-04 | manual | Push to GitHub, verify Docker workflow runs | N/A W0 | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [ ] `.golangci.yml` — golangci-lint v2 config file (does not exist yet)
- [ ] `.github/workflows/` — directory does not exist yet
- [ ] Verify `golangci-lint run` passes locally before committing workflow

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CI workflows trigger on push | CI-01, CI-02, CI-03 | Requires GitHub Actions runtime | Push to branch, check Actions tab |
| Docker images pushed to GHCR on main merge | CI-04 | Requires GitHub Actions + GHCR | Merge to master, check Packages tab |
| Coverage reported in CI output | CI-05 | Requires CI log inspection | Check Go CI workflow run logs |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
