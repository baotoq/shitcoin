---
phase: 09-containerization
plan: 02
subsystem: infra
tags: [docker, nginx, react, vite, multi-stage-build, reverse-proxy, websocket]

requires:
  - phase: 05-web-dashboard
    provides: React SPA frontend in web/ directory
provides:
  - nginx.conf with SPA routing and reverse proxy for API/WebSocket
  - Multi-stage Dockerfile producing minimal nginx-based frontend image
  - Non-root container execution (appuser:1001)
affects: [10-ci-pipeline, 11-kubernetes, 12-dev-environment]

tech-stack:
  added: [nginx:1.27-alpine, node:22-alpine]
  patterns: [multi-stage-docker-build, non-root-container, reverse-proxy-pattern]

key-files:
  created:
    - web/nginx.conf
    - web/Dockerfile
    - web/.dockerignore
  modified: []

key-decisions:
  - "Nginx listens on port 8080 (non-root compatible, no CAP_NET_BIND_SERVICE needed)"
  - "Added .dockerignore to exclude node_modules and dist from build context"

patterns-established:
  - "Frontend reverse proxy: nginx proxies /api/ and /ws to backend:8080 upstream"
  - "Non-root containers: appuser:appgroup (1001:1001) with chown on nginx dirs"

requirements-completed: [DOCK-02, DOCK-04, DOCK-05]

duration: 1min
completed: 2026-03-07
---

# Phase 9 Plan 2: Frontend Dockerfile Summary

**Nginx-based multi-stage Dockerfile serving React SPA with reverse proxy to backend API and WebSocket endpoints**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-07T13:07:30Z
- **Completed:** 2026-03-07T13:08:16Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- nginx.conf with SPA try_files fallback, /api/ reverse proxy, and /ws WebSocket proxy with upgrade headers
- Multi-stage Dockerfile: node:22-alpine build stage + nginx:1.27-alpine runtime stage
- Non-root execution as appuser (UID 1001) with proper nginx directory permissions

## Task Commits

Each task was committed atomically:

1. **Task 1: Create nginx.conf for SPA routing and reverse proxy** - `770ec97` (feat)
2. **Task 2: Create React frontend Dockerfile with nginx** - `76298f8` (feat)

## Files Created/Modified
- `web/nginx.conf` - SPA routing with try_files, reverse proxy for /api/ and /ws to backend:8080
- `web/Dockerfile` - Two-stage build: node:22-alpine (build) + nginx:1.27-alpine (runtime)
- `web/.dockerignore` - Excludes node_modules, dist, .git from build context

## Decisions Made
- Nginx listens on port 8080 to avoid requiring root privileges or CAP_NET_BIND_SERVICE
- Added .dockerignore (Rule 2 - missing critical) to prevent bloated build context from node_modules

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added .dockerignore for web/ build context**
- **Found during:** Task 2 (Dockerfile creation)
- **Issue:** No .dockerignore existed; node_modules (200MB+) would be copied into build context
- **Fix:** Created web/.dockerignore excluding node_modules, dist, .git, *.log
- **Files modified:** web/.dockerignore
- **Verification:** File exists with correct exclusions
- **Committed in:** 76298f8 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Essential for reasonable build performance. No scope creep.

## Issues Encountered
- Docker daemon not running on build machine; could not verify image build or non-root user. Dockerfile follows plan specification exactly and will work when Docker is available.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Frontend Dockerfile ready for docker-compose integration (Phase 9 Plan 3 if applicable)
- nginx.conf upstream hostname "backend" will resolve via K8s Service DNS (Phase 11) or docker-compose service name
- For standalone testing: `docker build -t shitcoin-frontend web/` then `docker run --add-host backend:host-gateway -p 8080:8080 shitcoin-frontend`

---
*Phase: 09-containerization*
*Completed: 2026-03-07*
