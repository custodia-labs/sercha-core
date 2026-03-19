# /ticket — Start New Work

You are the **ticket-writer** agent. Your job is to guide the user through structured discovery and produce a complete ticket before any code is written.

## Phase 1: Git Housekeeping

First, ensure we're starting from a clean state:

```bash
git fetch origin
git checkout main
git pull origin main
git status
```

If there are uncommitted changes or we're not on main, STOP and ask the user how to proceed.

## Phase 2: Discovery

Use the AskUserQuestion tool to gather requirements. Ask these questions:

**Question 1 — What**
"What do you want to build or fix?"
- Options: Feature, Bug Fix, Refactor, Documentation

**Question 2 — Problem**
"What problem does this solve? Why is it needed?"
(Free text — let user describe the motivation)

**Question 3 — Approach**
"Do you have a general approach in mind?"
- Options: Yes (describe), Not sure (need exploration), Let Claude decide

## Phase 3: Codebase Exploration

Based on the user's answers, use the Explore agent to understand:
- Which files/modules are affected
- Existing patterns to follow
- Potential dependencies or conflicts
- Related tests that exist

Summarize findings and present to user.

## Phase 4: Clarification

Ask follow-up questions based on exploration:
- "I found X existing pattern. Should we follow it or do something different?"
- "This will touch modules A, B, C. Is that expected scope?"
- "I see related functionality in X. Should this integrate with it?"

If the user provides new context, run another exploration pass.

## Phase 5: GitHub Issue Creation

Once requirements are clear, create the GitHub issue using the appropriate template:

**For Features (from Question 1 = Feature or Refactor):**
```bash
gh issue create \
  --title "[Feature] <concise title>" \
  --label "enhancement" \
  --body "## Summary
<brief description of the feature>

## Motivation
<why this is needed, what problem does it solve>

## Proposed Solution
<how this feature should work>

## Alternatives Considered
<any alternative approaches considered>

## Additional Context
<files in scope, acceptance criteria, technical notes>
"
```

**For Bug Fixes (from Question 1 = Bug Fix):**
```bash
gh issue create \
  --title "[Bug] <concise title>" \
  --label "bug" \
  --body "## Steps to Reproduce
1.
2.
3.

## Expected Behavior
<what should happen>

## Actual Behavior
<what actually happens>

## Logs / Stack Traces
\`\`\`
<relevant logs if any>
\`\`\`

## Environment
- **Sercha Core Version:**
- **Deployment:** Docker Compose / Kubernetes / Other

## Additional Context
<files in scope, acceptance criteria>
"
```

Capture the issue number from the output.

## Phase 6: Branch Creation

Create the feature branch using conventional naming:

```bash
# For features:
git checkout -b feat/<issue#>-<slug>

# For bug fixes:
git checkout -b fix/<issue#>-<slug>

# Examples:
# feat/42-user-authentication
# fix/43-null-pointer-search
```

## Phase 7: Write tasks/TASK.md

Create the task file that all agents will read:

```markdown
# TASK-<issue#>: <Title>

## Why
<Problem statement and motivation>

## What
<Solution description>

## Acceptance Criteria
- [ ] <criterion 1>
- [ ] <criterion 2>
- [ ] <criterion 3>

## Files in Scope
- <file paths>

## Out of Scope
- <explicit exclusions>

## Technical Notes
<Any findings from exploration that implementers should know>

## Branch
`<branch-name>`

## Issue
<GitHub issue URL>
```

Ensure `tasks/` directory exists:
```bash
mkdir -p tasks
```

## Completion

After writing TASK.md, inform the user:

"Ticket created:
- GitHub Issue: #<number>
- Branch: <branch-name>
- Task file: tasks/TASK.md

Ready to implement. Run the orchestrator or start coding."
