LANE: full

Files:
- .github/workflows/ci.yml (new)
- .golangci.yml (new)
- factory/factory.yaml (edit: add `lint` to `commands`)

Reasoning: The change is CI and tool config, which is not docs, test, or comment-only work, and the expected files (.github/workflows/ci.yml, .golangci.yml) do not match the trivial paths `*.md` or `.mex/**`. The workflow also needs a Postgres service container and a Go toolchain run, so it is outside the trivial lane.
