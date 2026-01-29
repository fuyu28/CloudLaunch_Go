# Repository Guidelines

## Project Structure & Module Organization
- `main.go` is the Wails entry point; backend logic lives in `internal/` (e.g., `internal/app`, `internal/services`, `internal/db`, `internal/storage`).
- `cmd/cloudlaunch` contains the CLI/app entry package layout used by the project.
- `frontend/` is the Vite + React UI (TypeScript, Tailwind, DaisyUI); built assets go to `frontend/dist` and are embedded by Wails.
- `build/` contains platform-specific packaging assets for Wails builds.
- `docs/` holds project notes and planning artifacts.
- `sqlc.yaml` configures SQLite code generation into `internal/db/sqlc`.

## Build, Test, and Development Commands
- `wails dev` runs the full app in dev mode (invokes Bun install/dev per `wails.json`).
- `wails build` produces production desktop builds using assets in `build/`.
- `bun install` installs frontend dependencies in `frontend/`.
- `bun run dev` starts the Vite dev server for UI-only work.
- `bun run build` builds the frontend bundle into `frontend/dist`.
- `bun run test` runs frontend tests with Vitest.
- `go test ./...` runs all backend Go tests.
- `sqlc generate` regenerates DB access code from `internal/db/queries` and `internal/db/migrations` (requires `sqlc`).

## Coding Style & Naming Conventions
- Go code should be `gofmt`-formatted (tabs, standard Go style). Keep package names short and lower-case; exported identifiers use `CamelCase`.
- Frontend code is TypeScript + React; follow existing file patterns in `frontend/` and keep formatting consistent with nearby files.
- Prefer small, focused packages in `internal/` and keep UI state/logic close to components in `frontend/`.

## Testing Guidelines
- Go tests follow `*_test.go` naming and live alongside packages (example: `internal/result/result_test.go`).
- Frontend tests use Vitest (`bun run test`); name files `*.test.ts(x)` or `*.spec.ts(x)` and place them near the components they cover.
- Cover new behavior with tests when it affects data handling, storage, or UI flows.

## Commit & Pull Request Guidelines
- Recent commits mostly follow Conventional Commit style with scopes (e.g., `feat(storage): ...`, `fix(db): ...`), though short imperative summaries are also used.
- When possible, use `feat(scope):` / `fix(scope):` for clarity; keep the subject concise and action-oriented.
- PRs should include a brief summary, testing notes/commands run, and screenshots or GIFs for UI changes.
