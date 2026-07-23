---
name: new-api-change-recipes
description: >-
  Repo-specific recipes for making code changes in this new-api fork: adding a
  toggleable setting end-to-end (env var + option + status API + admin settings
  UI), building and testing Go without a local toolchain (Docker), the backend
  controller test fixture pattern with its gotchas, and local frontend tooling
  quirks. Use for any backend/frontend feature or fix in this repository —
  especially new options/settings, registration/auth-flow changes, controller
  changes needing tests, or when `go`/`bun` are not installed locally.
---

# new-api Change Recipes

Practical, verified recipes for this repo. Complements `AGENTS.md` (rules) with
how-to knowledge (mechanics). For frontend i18n work, always defer to the
`i18n-translate` skill — locale writes go through its script, never hand-edits.

## Recipe 1: Add a toggleable option end-to-end

A new boolean setting (example: `InvitationCodeRegisterEnabled`) touches all of
these — missing any one leaves the feature half-wired:

**Backend**

1. `common/constants.go` — `var MyOptionEnabled = false`
2. `common/init.go` — env-var default:
   `MyOptionEnabled = GetEnvOrDefaultBool("MY_OPTION_ENABLED", false)`.
   Env only sets the initial default; a value in the `options` DB table always
   wins (loaded via `loadOptionsFromDatabase`). Say so in `.env.example`.
3. `.env.example` — document the variable (file is Chinese-commented; match it)
4. `model/option.go` — two places:
   - `initOptionMap`: `common.OptionMap["MyOptionEnabled"] = strconv.FormatBool(common.MyOptionEnabled)`
   - the bool switch in `updateOptionMap`: `case "MyOptionEnabled": common.MyOptionEnabled = boolValue`
5. `controller/misc.go` `GetStatus` — expose a snake_case key
   (`"my_option_enabled": common.MyOptionEnabled`) so the frontend can react.
6. Backend i18n if new error messages: key const in `i18n/keys.go`, text in
   `i18n/locales/{en,zh-CN,zh-TW}.yaml`. Use `common.ApiErrorI18n(c, i18n.MsgXxx)`.

**Frontend (`web/`)** — follow how `RegisterEnabled` is wired:

1. `src/features/system-settings/types.ts` — add to the settings type
2. `src/features/system-settings/<area>/index.tsx` — default-values mapping
3. `src/features/system-settings/<area>/section-registry.tsx` — pass into the section
4. The section component — zod schema + `SettingsSwitchItem` UI; the existing
   diff-submit loop usually needs no change
5. `src/features/auth/types.ts` — status type has **two** spots (top-level and
   nested `data`); add the field to both. Read with the established fallback:
   `status?.my_option ?? status?.data?.my_option ?? default`

## Recipe 2: Go build/test without a local toolchain

This machine has **no `go`/`gofmt` installed** and Docker daemon may be down
(`open -a OrbStack` starts it). Run everything in a container with cached
volumes so repeated runs are fast:

```bash
docker run --rm -v "$PWD":/src -w /src \
  -v newapi-gomodcache:/go/pkg/mod \
  -v newapi-gobuildcache:/root/.cache/go-build \
  golang:1.26 sh -c 'gofmt -l -w <dirs> && go test ./controller/ ./model/ ./common/'
```

Gotchas:

- `go build ./...` **fails**: `main.go` has `go:embed web/dist` and `web/dist`
  doesn't exist until a frontend build. Build packages instead
  (`go build ./controller/... ./model/...`), or `mkdir -p web/dist && touch web/dist/index.html` first.
- `go vet ./common/` has **pre-existing warnings** (custom-event.go lock copies,
  email_test.go IPv6) — not yours; confirm with `git stash` + re-run if blamed.
- Match the toolchain the repo uses: go.mod says 1.25.x, Docker uses golang 1.26.x.

## Recipe 3: Backend controller test fixture

Copy the pattern from `controller/auth_flow_test.go` and
`controller/register_invitation_test.go`:

- In-memory DB: `gorm.Open(sqlite.Open(":memory:"))` (`github.com/glebarez/sqlite`),
  `AutoMigrate` only the models you touch, then swap globals and restore in
  `t.Cleanup`: `model.DB`, `common.SetMainDatabaseType(...)`, and every
  `common.*` toggle your code path reads.
- **`common.RedisEnabled` defaults to `true`** with a nil client — any code path
  that calls `updateUserCache` (e.g. `User.Insert` → `finishInsert` → `Update`)
  **panics** unless you set `common.RedisEnabled = false` in the fixture. This
  is the #1 test-environment trap.
- Handlers using `common.ApiErrorI18n` need `i18n.Init()` once (sets
  `common.TranslateMessage`; safe across tests, uses `sync.Once`).
- Default test language resolves to `i18n.LangEn`; assert messages with
  `i18n.Translate(i18n.LangEn, key)` instead of hardcoding strings.
- Set `common.QuotaForNewUser = 0` in register-flow tests to skip the log-write
  path; `GenerateDefaultToken` defaults false.
- New tests MUST use `stretchr/testify` (`require` for setup/fatal, `assert`
  for values) per AGENTS.md.

## Recipe 4: Frontend tooling quirks

- **bun is not installed** — use `npx -y bun ...` (e.g. `npx -y bun run typecheck`).
  `bunx` resolution can fail; call binaries directly:
  `web/node_modules/.bin/oxlint <files>`.
- After TS/TSX changes: `npx -y bun run typecheck`, oxlint on touched files,
  and `npx -y bun run format` if `format:check` complains (it preserves
  copyright headers).
- Frontend status fields often arrive either flat or nested under `data` —
  mirror the `status?.x ?? status?.data?.x ?? fallback` pattern.

## Recipe 5: Shared endpoint gating (auth-dependent behavior)

When a public endpoint also serves logged-in flows (e.g. `/api/verification`
serves both sign-up codes and profile email binding), don't hard-block it on a
global toggle. Add `middleware.TryUserAuth()` to the route (sets `id` when
credentials exist, never rejects), then gate in the handler:
`if !common.SomeEnabled && c.GetInt("id") == 0 { reject }`. Precedent:
`/api/oauth/state` route and `SendEmailVerification`.

## Operational notes

- Fork workflow: `.github/workflows/sync-upstream.yml` merges upstream hourly
  (GitHub `merge-upstream` API), syncs tags, and builds/pushes
  `ghcr.io/qjx12388/new-api:latest` (matrix amd64/arm64 on native runners +
  `imagetools create` manifest) when main moved or a new tag appeared; human
  pushes to `main` also trigger the build (`actor != 'github-actions[bot]'`
  filter prevents the workflow's own merge push from double-building).
  Pushing any tag triggers the build directly from the tagged commit and
  additionally publishes `ghcr.io/qjx12388/new-api:<tag>`.
- Pushing commits that touch `.github/workflows/**` requires a PAT with the
  `workflow` scope (fine-grained: Repository permissions → Workflows: R/W).
- Git credentials are in macOS keychain (`osxkeychain` helper); verify with
  `printf "protocol=https\nhost=github.com\n" | git credential fill` — never
  print the password line.
