# /pr — Create Pull Request

Push the feature branch and create a PR using the appropriate template.

## Pre-flight Checks

1. Verify we're on a feature branch (not main):
```bash
git branch --show-current
```

If on main, STOP: "Cannot create PR from main. Switch to a feature branch first."

2. Verify review passed:
```bash
cat tasks/review-notes.md | grep "Recommendation"
```

If NEEDS FIXES, STOP: "Review found issues. Fix them before creating PR."

3. Check for uncommitted changes:
```bash
git status --porcelain
```

If changes exist, STOP: "Uncommitted changes found. Commit or stash before creating PR."

## Determine PR Type

Parse the branch name to determine type:
- `feat/*` → Feature PR (use `.github/PULL_REQUEST_TEMPLATE/feature.md`)
- `fix/*` → Bugfix PR (use `.github/PULL_REQUEST_TEMPLATE/bugfix.md`)

## Push Branch

```bash
git push -u origin $(git branch --show-current)
```

## Read TASK.md for Context

```bash
cat tasks/TASK.md
```

Extract:
- Issue number (from ## Issue section)
- Summary (from ## What section)
- Acceptance criteria

## Create PR

**For Feature PRs (`feat/*` branches):**
```bash
gh pr create \
  --title "feat: <title from TASK.md>" \
  --body "## Summary
<from TASK.md What section>

## Motivation
<from TASK.md Why section>

## Changes
- <list key changes made>

## Testing
- [x] Unit tests added/updated
- [x] Manual testing performed
- [ ] Tested Docker build

**Test command:** \`go test ./...\`

## Checklist
- [x] Code follows project conventions
- [x] Commits follow [Conventional Commits](https://www.conventionalcommits.org/)
- [ ] Documentation updated (if applicable)
- [ ] No breaking changes (or documented in commits with \`!\`)
- [ ] CI passes

## Related Issues
Closes #<issue number>
"
```

**For Bugfix PRs (`fix/*` branches):**
```bash
gh pr create \
  --title "fix: <title from TASK.md>" \
  --body "## Bug Description
<from TASK.md Why section>

## Root Cause
<from TASK.md Technical Notes or explain>

## Fix
<from TASK.md What section>

## Reproduction Steps
<from original issue or TASK.md>

## Testing
- [x] Bug no longer reproduces
- [x] Regression test added
- [x] Manual testing performed

**Test command:** \`go test ./...\`

## Checklist
- [x] Code follows project conventions
- [x] Commits follow [Conventional Commits](https://www.conventionalcommits.org/)
- [x] Root cause identified and documented
- [x] Tests added to prevent regression
- [ ] CI passes

## Related Issues
Fixes #<issue number>
"
```

## Completion

After PR is created, report:

"PR created:
- URL: <PR URL from gh output>
- Branch: <branch name> → main
- Issue: #<issue number> will be closed on merge

Next: Wait for CI, request review if needed."
