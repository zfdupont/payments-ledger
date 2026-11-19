LANE: full
Files:
- a new versioned migration adding append-only triggers (migration directory not yet confirmed)
- internal/ledger/append_only_test.go (new Go test file)

Reasoning: the change is a database migration plus a new Go test, which is neither docs-only, test-only, comment-only, nor config-only, and none of the expected files match lanes.trivial.paths (*.md, .mex/**).
