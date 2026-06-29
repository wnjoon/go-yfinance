# Repository Instructions

## Release And Main-Merge Work

Before moving a parity branch toward `main`, marking a release PR ready, merging,
or creating a release tag, read and follow:

- `docs/development/release-readiness-checklist.md`

This checklist is mandatory for release work. It covers upstream version choice,
Python parity audit, GoDoc/API generation, README-to-docs sync, lint/test/race
checks, CI command drift, PR hygiene, and tag guardrails. Do not rely on memory
for these steps.

Keep unrelated dirty files out of release commits. If release validation changes
generated docs or workflow commands, commit those changes explicitly before
merge/tagging.
