# /review — Code Review Checklist

You are the **go-reviewer** agent. Review all changed Go files and report issues before tests run.

## Step 1: Identify Changed Files

```bash
git diff --name-only main...HEAD -- '*.go'
```

If no changes, report "No Go files changed since main."

## Step 2: Run Automated Checks

```bash
go test ./...
golangci-lint run
go vet ./...
```

Report any failures immediately.

## Step 3: Dependency Violation Check

For each changed file, check for import violations:

**domain/ files must NOT import:**
- `internal/core/ports/`
- `internal/adapters/`

**services/ files must NOT import:**
- `internal/adapters/`

Use grep to check:
```bash
grep -l "adapters/" internal/core/domain/*.go internal/core/services/*.go 2>/dev/null
```

If violations found, HALT and report:
- File path
- Violating import
- Required fix

## Step 4: Test Coverage Check

For each new or modified file, verify corresponding tests exist:

```bash
# List changed source files (non-test)
git diff --name-only main...HEAD -- '*.go' | grep -v '_test.go'
```

For each file `foo.go`, check if `foo_test.go` exists and has relevant tests.

Report missing test coverage.

## Step 5: Swagger Annotation Check

If any files in `internal/adapters/driving/http/` changed:

```bash
git diff --name-only main...HEAD -- 'internal/adapters/driving/http/*.go'
```

Check if swagger annotations are present and suggest running `/swagger` if handlers were modified.

## Step 6: Error Handling Patterns

Scan for common issues:
- Ignored errors: `_ = someFunc()`
- Naked returns in functions with named return values
- Missing error wrapping context

## Output

Write findings to `tasks/review-notes.md`:

```markdown
# Review Notes — <date>

## Automated Checks
- Tests: PASS/FAIL
- Lint: PASS/FAIL
- Vet: PASS/FAIL

## Dependency Violations
<list or "None">

## Missing Tests
<list or "None">

## Swagger Status
<current or needs update>

## Other Issues
<list or "None">

## Recommendation
<READY TO MERGE / NEEDS FIXES>
```

Inform the user of the recommendation.
