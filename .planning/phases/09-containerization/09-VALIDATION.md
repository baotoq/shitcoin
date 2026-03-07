---
phase: 9
slug: containerization
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-07
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Docker CLI (docker build, docker run, docker exec) |
| **Config file** | Dockerfile, web/Dockerfile |
| **Quick run command** | `docker build -t shitcoin-backend .` |
| **Full suite command** | `docker build -t shitcoin-backend . && docker build -t shitcoin-frontend web/` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `docker build -t shitcoin-backend .`
- **After every plan wave:** Run `docker build -t shitcoin-backend . && docker build -t shitcoin-frontend web/`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 1 | DOCK-03 | manual | `cat .dockerignore` | N/A W0 | pending |
| 09-01-02 | 01 | 1 | DOCK-01 | smoke | `docker build -t shitcoin-backend . && docker image inspect shitcoin-backend --format '{{.Size}}'` | N/A W0 | pending |
| 09-01-03 | 01 | 1 | DOCK-05 | smoke | `docker run --rm shitcoin-backend whoami` | N/A W0 | pending |
| 09-02-01 | 02 | 1 | DOCK-04 | manual | `cat web/nginx.conf` | N/A W0 | pending |
| 09-02-02 | 02 | 1 | DOCK-02 | smoke | `docker build -t shitcoin-frontend web/` | N/A W0 | pending |
| 09-02-03 | 02 | 1 | DOCK-05 | smoke | `docker run --rm shitcoin-frontend whoami` | N/A W0 | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

None -- validation uses Docker CLI directly, no test framework setup needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| .dockerignore content | DOCK-03 | File content check | Verify .dockerignore contains data/, wallets.json, .git, node_modules entries |
| nginx.conf correctness | DOCK-04 | Config syntax verification | Verify nginx.conf has try_files, /api proxy_pass, /ws with Upgrade headers |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
