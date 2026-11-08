# Row 3: Options.Tui + fallback chain + style build (internal/picker ONLY)
Worktree: /Users/khanghua/Workspace/github.com/khanghua1505/kubie-wt/tui-row3 (branch lane/row3, base 3bdfb89). Plan row 3 lives in MAIN checkout .omo/plans/tui-default-drop-fzf.md - READ ONLY, never touch.
FIRST: cd /Users/khanghua/Workspace/github.com/khanghua1505/kubie-wt/tui-row3 && git status && pwd - confirm worktree. NEVER touch /Users/khanghua/Workspace/github.com/khanghua1505/kubie (main checkout) or kubie-wt/tui-row2, tui-row6, tui-row7.

## Changes
1. internal/picker/picker.go:
   - Options struct: DELETE Mode, ModeAuto, ModeTerminal, ModeTui constants + `Mode Mode` field. Replace with `Tui TuiSettings` field (import github.com/khanghua1505/kon/internal/settings).
   - SelectWithOptions fallback chain, exact order:
     a) len(items)==1 -> return items[0], nil immediately (no TUI).
     b) o.Tui.IsEnabled() && isTTY(in) && isTTY(out) -> runTui(...) with newTuiModel.
     c) else if isTTY(out) -> promptList(items) (existing loop).
     d) else -> listItems(items) numbered non-TTY list.
   - Add seam var in terminal.go (or picker.go): `var isTTY = func(f *os.File) bool { fi, err := f.Stat(); return err == nil && fi.Mode()&os.ModeCharDevice != 0 }`.
   - TUI path: sel, border := o.Tui.ColorValues(); res1 := style.Resolve(sel); res2 := style.Resolve(border) - each on error fall back to defaults (style.Resolve already falls back internally to 212/238; if it returns (string, bool) treat false as use default). Check internal/tui/style API and use it as-is.
   - promptList keeps prompt item> / list plain> labels as today.
2. internal/picker/picker_tui.go: model constructor signature newTuiModel(items []string, sel, border lipgloss.Style or string) per existing code; runTui uses Program with In/Out from options. Adapt existing code minimally.
3. Style wiring: replace hardcoded 212/238 defaults with resolved values from o.Tui.ColorValues().

## Acceptance (must all pass)
- go build ./internal/... exit 0
- go test -count=1 ./internal/picker/ exit 0
- grep -rn "Mode" internal/picker/ matches ONLY test files (constants gone from product code)
- grep -rn "fzf" internal/picker/ -i: zero matches

## Evidence
mkdir -p .mo/evidence && save all acceptance command outputs to .mo/evidence/task-3.txt

## Commit
GIT_MASTER=1 git add -A && GIT_MASTER=1 git commit -m 'feat(picker): TUI default picker with fallback chain'
NO PUSH. DoneClaim = commit hash + acceptance summary. Root re-verifies.
