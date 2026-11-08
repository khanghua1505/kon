# Row 6: README update
Worktree: kubie-wt/tui-row6 (branch lane/row6, base = row-2 commit; root confirms). Plan row 6 in main-checkout .omo/plans/tui-default-drop-fzf.md (READ ONLY).
GOAL: README.md reflects new reality: fzf gone, TUI default picker, config block for tui.
DO: delete every fzf mention/section (search: fzf, FZF, FZF_DEFAULT_COMMAND) - lines around 22, 67, 82-84, 175-205 known; replace picker section prose with: kon ships built-in TUI picker (bubbletea), enabled by default; tui: false disables -> falls back to terminal list; tui: {selected_color, border_color} accepts 16 names (black..brightwhite), ANSI 256 decimal, #RRGGBB; invalid color falls back to default (212/238) with no error; stale fzf: key ignored. Config example block updated. No new deps, README only.
ACCEPTANCE: grep -in fzf README.md = 0 matches; go build ./... exit 0 (README-only change must not break build).
EVIDENCE: outputs to worktree .omo/evidence/ulw/tui-default-drop-fzf/a1/task-6-tui-default-drop-fzf.txt (mkdir -p).
COMMIT: GIT_MASTER=1 git add -A && git commit -m 'docs(readme): document TUI default picker, drop fzf' (lane/row6, no push).
DONECLAIM: commit hash + acceptance outputs + evidence path. Root re-verifies.
