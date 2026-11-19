LANE: full

Files:
- docker-compose.yml

Reasoning: The job creates a new `docker-compose.yml` at the repo root, which is outside `lanes.trivial.paths` (`*.md`, `.mex/**`) and matches the `escalate` entry for `docker-compose.yml` (build toolchain / dependencies), so it cannot be trivial.
