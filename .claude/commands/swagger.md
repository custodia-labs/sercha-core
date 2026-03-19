# /swagger — Regenerate Swagger Documentation

Regenerate the Swagger/OpenAPI documentation from handler annotations.

## Steps

1. Run swag init:
```bash
swag init -g cmd/sercha-core/main.go -o docs
```

2. Verify the output files were updated:
```bash
ls -la docs/swagger.json docs/swagger.yaml
```

3. Show a summary of changes:
```bash
git diff --stat docs/
```

4. If there are changes, inform the user:
   - "Swagger docs updated. Review changes in docs/ before committing."

5. If no changes:
   - "Swagger docs are already up to date."
