# Row 2: TuiSettings dual-form decode
Worktree: /Users/khanghua/Workspace/github.com/khanghua1505/kubie-wt/tui-row2 (branch lane/row2, base a8ee047). Plan row 2: /Users/khanghua/Workspace/github.com/khanghua1505/kubie/.omo/plans/tui-default-drop-fzf.md (READ ONLY).
GOAL: settings.Tui becomes struct TuiSettings { Enabled *bool; SelectedColor *string; BorderColor *string } with custom yaml.Unmarshal handling BOTH forms: scalar bool (tui: true/false -> Enabled set, colors nil) AND map (tui: {selected_color: "pink", border_color: "238"}). Unknown map keys ignored; stale fzf: key at settings root silently ignored. Defaults applied in Settings defaults fn: Enabled default true, SelectedColor "212", BorderColor "238". Drop Settings.Fzf field + fzf handling from decode.
FILES (expected): internal/settings/settings.go:24,29,60-84; internal/settings/decode.go:37,79-80,209-225; internal/settings/settings_test.go:16-17,28-30,164-171,195-197,210-232.
TESTS: settings_test.go - delete fzf asserts 16-17, 219-232; rewrite bool asserts `if s.Tui` (28-30) -> `if !s.Tui.Enabled`, `if !s.Tui` (195-197) -> `if s.Tui.Enabled`; add table tests: scalar true/false; map partial (only selected_color) keeps enabled true; INVALID color "neon" loads fine (NO error - user decision), falls back to defaults 212/238 at accessor/default site; stale fzf: key ignored.
SCOPE: internal/settings ONLY. cmd/ compile break after this change is EXPECTED (cmd/ctx.go `if cfg.Tui` breaks) - do NOT touch cmd/. go build ./cmd/ failing is not a row-2 failure.
ACCEPTANCE: go build ./internal/... exit 0; go test -count=1 ./internal/settings/ green incl new table tests.
EVIDENCE: outputs to .omo/evidence/ulw/tui-default-drop-fzf/a1/task-2-tui-default-drop-fzf.txt (mkdir -p).
COMMIT: GIT_MASTER=1 git add -A && git commit -m 'feat(settings): TuiSettings dual-form decode, drop fzf settings' (lane/row2, no push).
DONECLAIM: commit hash + acceptance outputs + evidence path. Root re-verifies.
