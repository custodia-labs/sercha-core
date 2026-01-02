## Release Version

<!-- Which version is being released? e.g., 0.1.0 -->

## Changes in This Release

### Features
-

### Bug Fixes
-

### Other Changes
-

## Pre-Release Checklist

- [ ] All CI checks pass
- [ ] Docker build tested locally: `docker build -t sercha-core:test .`
- [ ] Manual testing of key features
- [ ] Version follows [Semantic Versioning](https://semver.org/)

## Release Checklist

- [ ] VERSION file updated to new version
- [ ] No other file changes in this PR
- [ ] Release notes prepared (above)

## Post-Merge

After this PR is merged:
1. GitHub Actions will create tag `v<VERSION>`
2. Docker image will be built and pushed to `ghcr.io/custodia-labs/sercha-core`
3. GitHub Release will be created with auto-generated notes

## Approvals

This PR requires owner/maintainer approval (CODEOWNERS).
