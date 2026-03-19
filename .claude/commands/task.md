# /task — Spawn Agent Team from Approved TASK.md

You are the **orchestrator** agent. Your job is to read the approved `tasks/TASK.md` and coordinate the implementation team.

## Pre-flight Checks

1. Verify TASK.md exists and is approved:
```bash
cat tasks/TASK.md
```

If missing, tell the user to run `/ticket` first.

2. Confirm with the user:
"I've read TASK.md. Ready to spawn the agent team for: **<task title>**

Files in scope:
- <list from TASK.md>

Proceed with implementation?"

## Agent Dispatch Order

### Stage 1: Domain (sequential)

Spawn **domain-expert** if TASK.md mentions:
- New domain models
- Changes to `internal/core/domain/`
- New aggregates or value objects

**domain-expert:**
- Reads: `tasks/TASK.md`
- Owns: `internal/core/domain/**`
- Outputs: `tasks/domain-contracts.md`
- Rule: NO imports from ports/ or adapters/

### Stage 2: Application Services & Ports (sequential, after Stage 1)

Spawn **app-svc** for:
- `internal/core/services/**`
- `internal/core/ports/**`

**app-svc:**
- Reads: `tasks/TASK.md`, `tasks/domain-contracts.md`
- Owns: `internal/core/services/**`, `internal/core/ports/**`
- Outputs: `tasks/port-interfaces.md`
- Rule: defines ports, never implements them; no adapter imports

### Stage 3: Adapters + UI (parallel, after Stage 2)

Once `tasks/port-interfaces.md` exists, spawn in parallel:

**adapter-impl:**
- Reads: `tasks/TASK.md`, `tasks/port-interfaces.md`
- Owns: `internal/adapters/**` (postgres, vespa, redis, connectors, http)
- Rule: implements ports, never defines business logic; can own migrations

**ui-agent:**
- Reads: `tasks/TASK.md`, `tasks/port-interfaces.md`
- Owns: `ui/**`
- Rule: never touches internal/; uses port-interfaces.md for API shape

### Stage 4: Testing (sequential, after Stage 3)

Spawn **tester-strict** after implementation:
- Reads: all changed source files, `tasks/TASK.md`
- Owns: `**/*_test.go` (write only)
- Rule: read-only on source, validates acceptance criteria

## Coordination Rules

1. **Sequential stages**: Domain → App/Ports → Adapters+UI → Tests
2. **Parallel within Stage 3**: adapter-impl and ui-agent run simultaneously
3. **Halt on violations**: If any agent detects import violations, stop and report
4. **File ownership**: Only one agent edits a file at a time
5. **Handoff via files**: Each stage reads upstream coordination files

## Spawning Agents

Use the Task tool for each agent with clear instructions:

```
domain-expert prompt:
"Read tasks/TASK.md. Implement domain changes in internal/core/domain/.
Write tasks/domain-contracts.md documenting new/modified types.
Rule: NO imports from ports/ or adapters/. Halt if violation detected."

app-svc prompt:
"Read tasks/TASK.md and tasks/domain-contracts.md.
Implement services in internal/core/services/.
Define ports in internal/core/ports/.
Write tasks/port-interfaces.md documenting port definitions.
Rule: define ports only, never implement adapters."

adapter-impl prompt:
"Read tasks/TASK.md and tasks/port-interfaces.md.
Implement adapters in internal/adapters/ that satisfy the port interfaces.
Rule: implement ports, never define business logic."

ui-agent prompt:
"Read tasks/TASK.md and tasks/port-interfaces.md.
Implement UI changes in ui/.
Use port-interfaces.md for API shape. Never touch internal/."

tester-strict prompt:
"Read all changed Go files and tasks/TASK.md.
Write tests in **/*_test.go.
Validate acceptance criteria from TASK.md.
Rule: read-only on source files, write only test files."
```

## Completion

After all agents complete:

1. Run `/review` to check for issues
2. Report status to user:
   - Files changed
   - Tests added
   - Any issues found

"Implementation complete. Run `/review` to verify, then commit when ready."
