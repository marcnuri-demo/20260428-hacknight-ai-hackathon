# Project ground rules

This repo is a fork of the AI Hacknight challenge. The upstream repo is the source of truth; we never open PRs upstream. All work happens on `origin` (the fork).

## Repository layout

- One directory per AI-built Unix tool: `ai_grep/`, `ai_sed/`, `ai_sort/`, `ai_uniq/`, `ai_wc/`.
- Each tool ships as a Docker image with the same name as its directory and reads stdin / writes stdout.
- `server.log` is the shared input fixture. The native Unix pipeline is the ground truth; AI versions must produce byte-identical output via `diff`.
- `README.md` is upstream-owned. Do not edit it — record local decisions in this file instead.

## Workflow

- Each problem is broken into isolated issues that can be picked up by independent coding agents. An issue must be self-contained: a single tool or a single problem's pipeline, with a clear pass/fail check.
- One branch per issue, one PR per branch. PRs target `main` on the fork (`origin`).
- Issues will be drafted as follow-up tasks in this session — no GitHub issue template is required.

## Acceptance criteria & definition of done

Every AI-tool issue must include **black-box behavioral tests** in its acceptance criteria.

**No mocks. No stubs. No fakes.** The tool is treated as an opaque container: tests pipe real input through the actual built binary (or container image) and through the real native Unix tool, then assert the outputs are byte-identical. Tests must not import the tool's internal Go packages, must not stub `os/exec`, must not substitute fake regex engines, and must not patch the native tool's behavior. If a test would need a mock to pass, it is the wrong test.

An issue is "done" only when **all** of the following are true:
1. The tool's source builds (`go build ./...`) and unit tests pass (`go test ./...`).
2. The container image builds.
3. **Black-box behavioral tests pass with no mocks** — every flag and input scenario listed in the issue's AC produces output byte-identical to the native tool, exercised end-to-end through the built artifact. These tests live with the tool (e.g., `ai_grep/test/`) and run in CI.
4. The end-to-end validation `diff` from the relevant problem in `README.md` produces empty output.
5. PR is opened, CI is green, squash-merged to `main`.

The black-box tests are the contract — they outlive the issue and protect against regressions when other tools or pipelines change. Mocked tests would defeat that contract by letting bugs hide behind faked dependencies.

## Implementation language

- All AI tools are implemented in **Go**.
- Reasons: static binaries, trivial cross-compilation across platforms, small final Docker images (scratch/distroless), strong stdlib for I/O and regex.
- Each tool is a single `main` package that reads stdin and writes stdout, mirroring the Unix tool's interface.

## Container runtime

- Use **Podman**, not Docker, for all local builds and runs. The `README.md` examples use `docker` because the upstream is the source of truth — locally, substitute `podman` (the CLI is drop-in compatible for `build` and `run`).
- Example: `podman build -t ai_grep ai_grep` and `podman run --rm -i ai_grep "ERROR"`.

## Build & module layout

- **One Go module per tool.** Each tool directory (`ai_grep/`, `ai_sed/`, …) has its own `go.mod`, `main.go`, and `Dockerfile`. No shared code or workspace at the repo root.
- Rationale: keeps each issue fully self-contained — an agent working on one tool never touches another tool's directory, eliminating cross-tool merge conflicts.
- **Multi-stage Dockerfile** per tool: `golang:<version>-alpine` builder stage that produces a static binary, copied into a `scratch` (or `distroless/static`) final stage. Final images stay tiny and contain only the binary.

## Flag parsing

- Use **`github.com/spf13/pflag`** for flag parsing in every tool.
- Reasons: handles GNU-style bundled short flags (`grep -oE`, `sort -rn`), familiar to the user, drop-in replacement for stdlib `flag`.
- For `sed`, the script is a positional argument (`sed 's/foo/bar/'`) — that's `pflag.Args()` after parsing, not a flag.
- Each tool only needs to implement the flags actually used by the pipelines in `README.md`. Problem 4's behavioral clone is the only place where wider flag coverage matters; that issue will spell out which flags are in scope.

## PR & merge rules

- **Merge strategy: squash and merge.** One commit per issue lands on `main`.
- **Required to merge:** the GitHub Actions pipelines must pass. No merging on red.
- All other rules (branch protection, required reviewers, required status checks, etc.) are configured directly in GitHub repository settings, not in this file.

## CI / GitHub Actions

Two workflows:

### PR workflow (runs on every pull request, blocks merge)
1. **Per-tool build & test** — for each `ai_*/` directory: `go build ./...`, `go test ./...`, then build the container image.
2. **End-to-end pipeline validation** — run the validation `diff` from each currently-implemented problem in `README.md` against `server.log`, using the just-built images. Empty diff = pass.

Build all images and validate all implemented problems on every PR — Go builds are cheap and this catches cross-tool regressions.

CI runs on GitHub Actions Ubuntu runners which are Docker-native, so the workflow uses `docker` even though local development uses `podman`. The CLI is compatible for `build` and `run`.

### Main workflow (runs on push to `main` after squash-merge)
3. **Build & push images to GHCR** — push each `ai_*` image to `ghcr.io/marcnuri-demo/20260428-hacknight-ai-hackathon/<tool>` (or equivalent path). Tag with both `latest` and the commit SHA.

