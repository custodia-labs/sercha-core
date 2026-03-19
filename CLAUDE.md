# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Sercha Core is a self-hosted, enterprise-grade search platform built in Go. It provides unified search across multiple data sources (GitHub, GitLab, Slack, Notion, Confluence, etc.) using hexagonal (ports & adapters) architecture.

**Tech Stack:** Go 1.24+, PostgreSQL 16+, Vespa 8+ (search), Redis (cache/queue), Next.js 15, React 19, TypeScript 5.7

## Commands

```bash
# Build & Run
go build -o sercha-core ./cmd/sercha-core
go run ./cmd/sercha-core all|api|worker

# Test & Lint
go test ./...
go test -cover ./...
golangci-lint run
go vet ./...

# Swagger
swag init -g cmd/sercha-core/main.go -o docs

# UI (in /ui)
npm run dev|build|test|lint:fix|typecheck

# Docker
docker build -t sercha-core:latest .
cd examples/quickstart && docker compose --profile ui up -d
```

## Architecture

```
internal/
├── core/
│   ├── domain/      # Domain models — NO imports from ports/ or adapters/
│   ├── ports/       # Interface definitions (driven/ and driving/)
│   └── services/    # Business logic, imports ports, never adapters
├── adapters/
│   ├── driven/      # postgres/, vespa/, redis/, connectors/, auth/
│   └── driving/     # http/ — REST handlers with Swagger annotations
├── worker/          # Background task processing
└── runtime/         # Dynamic service config (AI services)
```

**Dependency Rule:** `domain ← ports ← adapters` — violations must be caught and fixed.

---

## Development Workflow

### Starting Work: `/ticket`

All feature work begins with `/ticket`. This command:

1. **Git sync** — pulls main, ensures clean working state
2. **Discovery** — asks What, Why, How; explores codebase; clarifies
3. **GitHub issue** — creates issue with full context
4. **Branch** — creates `feat/<issue#>-<slug>` or `fix/<issue#>-<slug>`
5. **TASK.md** — writes `tasks/TASK.md` with acceptance criteria

Never start coding without an approved `tasks/TASK.md`.

### During Work

- Run tests frequently: `go test ./...`
- Check imports: no adapter imports in domain/services
- Update swagger if HTTP handlers change: `/swagger`

### Before Commit

Commits are blocked unless:
- All tests pass
- Linter passes
- New functionality is manually verified
- Bug fixes are confirmed working

**Commit format:** Conventional Commits only, no Claude authorship.
```
type(scope): description

# NO Co-Authored-By, NO 🤖 Generated with Claude Code
```
- **Types:** feat, fix, docs, style, refactor, test, chore
- **Scopes:** core, adapters, http, worker, ui, docs, ci
- **Branch → Commit alignment:** `feat/42-user-auth` → `feat(core): add user authentication`

### Code Review: `/review`

Before marking work complete, run `/review` to check:
- Import violations (adapter in core/)
- Missing tests for new code
- Swagger annotations current
- Error handling patterns

### Create PR: `/pr`

After review passes, run `/pr` to:
- Push feature branch to origin
- Create PR using correct template (feature.md or bugfix.md)
- Link to GitHub issue

---

## Agent Architecture

### Agent Roster

| Agent | Model | Owns |
|-------|-------|------|
| ticket-writer | opus | Discovery, GitHub issue, tasks/TASK.md |
| orchestrator | opus | Coordinates implementation, reads TASK.md |
| domain-expert | sonnet | internal/core/domain/** |
| app-svc | sonnet | internal/core/services/**, internal/core/ports/** |
| adapter-impl | sonnet | internal/adapters/** |
| ui-agent | sonnet | ui/** (reads port-interfaces.md for API shape) |
| tester-strict | sonnet | **/*_test.go (read-only on src, write tests only) |
| go-reviewer | sonnet | Read-only; writes tasks/review-notes.md only |

### Pipeline

```
/ticket → [approve TASK.md] → /task → /review → commit → /pr
```

**Implementation stages (`/task`):**
```
Stage 1: domain-expert       → tasks/domain-contracts.md
Stage 2: app-svc             → tasks/port-interfaces.md
Stage 3: adapter-impl + ui-agent (parallel)
Stage 4: tester-strict
```

### Coordination Files

```
tasks/
├── TASK.md              # Current ticket: What, Why, Acceptance Criteria, Files in Scope
├── domain-contracts.md  # Domain models and events (domain-expert output)
├── port-interfaces.md   # Port definitions adapters must implement
└── review-notes.md      # Issues found before merge
```

### Agent Rules

- **domain-expert:** never imports from ports/ or adapters/
- **app-svc:** defines ports in internal/core/ports/, never implements them
- **adapter-impl:** implements ports, never defines business logic; can own migrations
- **ui-agent:** never touches internal/; reads port-interfaces.md for API shape
- **tester-strict:** read-only on source, writes only test files
- **go-reviewer:** read-only on all source; writes only tasks/review-notes.md
- **All agents:** declare file edits before writing; halt on dependency violations

### Expanding the Roster

Split `adapter-impl` into separate agents (adapter-postgres, adapter-vespa, adapter-http) when:
- Two adapters change simultaneously on the same ticket
- Parallel agent teams become necessary

---

## Slash Commands

| Command | Purpose |
|---------|---------|
| `/ticket` | Start new work: git sync → discovery → GitHub issue → branch → TASK.md |
| `/task` | Spawn agent team from approved tasks/TASK.md |
| `/swagger` | Regenerate swagger docs |
| `/review` | Run go-reviewer checklist on changed files |
| `/pr` | Push branch and create PR using template |

---

## Hooks

Configured in `.claude/settings.json`. Post-edit hook runs after Go file changes:

- `go vet ./...`
- `go test ./...`
- `golangci-lint run`

Exit code 2 blocks the action if any check fails.

**Scope:** Hook only fires on `*.go` file edits — not ui/, docs/, or migrations.

---

## Environment Variables

```bash
DATABASE_URL         # PostgreSQL connection (required)
REDIS_URL            # Redis (optional, falls back to Postgres queue)
JWT_SECRET           # JWT signing secret
VESPA_CONFIG_URL     # Default: http://localhost:19071
VESPA_CONTAINER_URL  # Default: http://localhost:8080
RUN_MODE             # api, worker, or all (default: all)
```

## Development Setup

```bash
cd examples/dev && docker compose up -d postgres vespa
go run ./cmd/sercha-core all
```

---

## Claude Code Web

For cloud-based sessions at claude.ai/code:

1. **Connect GitHub** — OAuth + install Claude GitHub app
2. **Configure environment** — Settings → Environment → Setup script: `.claude/setup.sh`
3. **Network access** — Set to "limited" (allows go mod download, npm install)

The setup script installs: Go tools (golangci-lint, swag), Go modules, and npm dependencies.

**Limitations vs CLI:**
- GitHub only (no GitLab)
- SessionStart hooks only (post-edit hooks won't run)
- Pull web sessions to CLI with `claude --teleport`

---

## Fork-Only Files

**Never stage `.claude/`, `tasks/`, or `CLAUDE.md` on feature branches.**

These files are tracked on this fork's main branch only. They must not appear in upstream PRs.

```bash
# Before creating PR, ensure these are not staged:
git reset HEAD .claude/ tasks/ CLAUDE.md
```
