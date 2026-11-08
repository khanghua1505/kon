# Row 4: wire all picker callers to Options{Tui, In, Out}
Worktree: /Users/khanghua/Workspace/github.com/khanghua1505/kubie-wt/tui-row4 (branch lane/row4, base 570ef95). Plan lives in MAIN checkout - never touch. FIRST: cd worktree && git status.
Scope: cmd/ ONLY (internal/ already done).

## Current broken state
Row 3 deleted picker.Mode/Modes; settings.Settings.Tui is now settings.TuiSettings (bool field gone). go build ./cmd/... fails at 3 sites.

## Changes (exactly 3 sites)
1. cmd/ctx.go ~line 73: DELETE the `mode := picker.ModeAuto; if cfg.Tui { mode = picker.ModeTui }` block. Call becomes:
   picker.SelectWithOptions(names, picker.Options{
       Tui: cfg.Tui,
       In:  cmd.InOrStdin(),
       Out: cmd.OutOrStdout(),
   })
2. cmd/ns.go ~line 172: replace `Mode: picker.ModeAuto,` with `Tui: <settings>.Tui,` - find the settings/config variable in scope of that function (grep upward for the func signature; pass it down if missing).
3. cmd/edit.go ~line 162 (func selectContext(cmd, installed, config *settings.Settings)): replace `Mode: picker.ModeAuto,` with `Tui: config.Tui,`.
delete.go: no picker call - leave alone.
Keep gofmt alignment (gofmt -l cmd/ = empty).

## Acceptance
- go build ./... exit 0
- go test -count=1 ./cmd/... ./internal/... all ok
- go vet ./... exit 0
- grep -rn "Mode" cmd/ --include='*.go' → zero matches

## Evidence
mkdir -p .mo/evidence; save outputs to .mo/evidence/task-4.txt

## Commit
GIT_MASTER=1 git add -A && GIT_MASTER=1 git commit -m 'feat(cmd): wire picker callers to TUI settings'
NO PUSH. DoneClaim = commit hash + acceptance summary.
