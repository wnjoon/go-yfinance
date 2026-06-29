# Release Readiness Checklist

Use this checklist before moving a parity branch to `main`, marking a PR ready,
or creating a release tag. It exists to prevent the v1.4.x class of mistakes:
missing upstream patch scope, generated-doc drift, lint/CI command drift, or tag
names that do not match the actual upstream parity target.

## 1. Version Decision

- Verify the upstream Python yfinance release page and tag status before choosing
  the Go tag.
- If an upstream release was retracted or superseded, do not create a matching Go
  tag just to keep version numbers contiguous. Go module tags do not need every
  patch number to exist.
- Document the decision in the release notes and progress document.
- For the 1.5.x line, Python yfinance `1.5.0` was retracted and `1.5.1` is the
  corrected parity target. Therefore go-yfinance should release `v1.5.1` without
  creating `v1.5.0`.

## 2. Upstream Parity Audit

- Review the upstream release notes, full changelog PR, and linked PRs/commits.
- For every upstream item, record one of:
  - implemented;
  - intentionally not applicable to Go;
  - deferred with a reason.
- Compare public API shape, not only internal behavior. Check method names,
  parameters, return semantics, model fields, defaults, and error behavior.
- Search for already-existing Go model fields that are not populated by the new
  implementation. The `Info().TrailingPegRatio` miss in v1.5.1 is the example to
  remember.
- Add regression tests for sparse, malformed, empty, and missing Yahoo payloads
  when upstream patches are defensive parsing fixes.

## 3. Local Test And Lint Gates

Run these from the repository root before merge or tagging:

```bash
GOCACHE=/tmp/go-build-cache go test ./...
GOCACHE=/tmp/go-build-cache go vet ./...
GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache golangci-lint run ./...
make lint
```

For release candidates, also run the CI-equivalent race and coverage test:

```bash
GOCACHE=/tmp/go-build-cache go test -v -race -coverprofile=coverage.out ./...
```

If integration tests fail only because the sandbox cannot resolve or reach Yahoo
or the Go module proxy, rerun the exact command in a network-enabled environment
and record that distinction in the progress document.

## 4. Documentation And GoDoc Gates

Regenerate and build docs whenever public APIs, exported comments, release notes,
or README content change:

```bash
GOMARKDOC=/tmp/gobin/gomarkdoc make docs
cp README.md docs/index.md
make docs-build
git diff --check
```

Confirm these files are in sync before merge:

- `README.md`
- `docs/index.md`
- `docs/API.md`
- `docs/api/*.md`
- `docs/releases/RELEASE_NOTES_vX.Y.Z.md`
- `docs/development/vX.Y.Z-progress.md`
- `mkdocs.yml`

MkDocs warnings about local `CLAUDE.md` files not being in nav are acceptable if
the site still builds. New warnings about broken release links, missing pages, or
unexpected nav omissions must be fixed.

## 5. CI And Tooling Drift

- Check `.github/workflows/test.yml` and `.github/workflows/docs.yml` against the
  local commands above.
- If a local tool version changes behavior, update the workflow or Makefile
  rather than relying on a command that only works in one environment.
- For `golangci-lint` v2, prefer package patterns that are verified locally and
  in CI. This repository has no Go files at the module root, so lint commands
  must actually analyze `./...`.

## 6. Git And PR Hygiene

- Keep unrelated dirty files out of release commits. In the v1.5.1 work,
  `pkg/repair/CLAUDE.md` was unrelated and intentionally left unstaged.
- Commit final verification/documentation fixes separately so the release audit
  trail is visible.
- Push the branch and verify the PR is mergeable:

```bash
git status --short --branch
gh pr view <number> --json number,state,isDraft,mergeable,title,url,headRefOid
```

## 7. Tagging Guardrail

Only tag after the branch is merged to `main` and the final commit on `main` is
the intended release commit. Before creating the tag:

- verify the version in release notes and install snippet;
- verify `mkdocs.yml` nav points to the new release notes/progress page;
- verify GoDoc/API docs are generated from the final code;
- verify CI checks pass on the merge commit;
- refresh or check Go Report Card after the tag/default branch is visible.
