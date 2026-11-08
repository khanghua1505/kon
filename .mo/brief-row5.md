# Row 5: picker fallback-chain tests
Worktree: /Users/khanghua/Workspace/github.com/khanghua1505/kubie-wt/tui-row5 (branch lane/row5, base 570ef95). Plan lives in MAIN checkout - never touch. FIRST: cd worktree && git status.
Scope: internal/picker tests ONLY (product code done in row 3; do NOT touch product files unless a test exposes a real bug - then report instead of fixing silently).

## Context (row 3 landed)
- Options{Tui settings.TuiSettings; In io.Reader; Out io.Writer} - Mode enum deleted.
- Fallback chain in SelectWithOptions: len==1 → Selected; Tui.IsEnabled() && attachedToTerminal → runTui; isTerminal(Out) → promptList; else listItems.
- Test seams present in picker_test.go: forceTTY(t) (overrides isTTY var) and promptOut(t) (pipe-backed *os.File Out + collect func). Copy those helpers' style.
- resolveColors(text) in picker.go maps settings colors via style.Resolve with 212/238 fallback.
- newTuiModel(items, selected, border lipgloss.TerminalColor) + newTestModel helper in picker_tui_test.go.

## Add tests (table or individual, follow existing test style)
1. Single item with Tui enabled + forced TTY + *os.File In/Out → still Selected immediately (chain a precedes TUI).
2. Tui enabled but In is not *os.File (strings.Reader) → promptList branch (menu contains "1) ").
3. Tui disabled + forced TTY Out → promptList branch.
4. No TTY at all (plain strings.Builder Out) → listItems branch (Listed, sorted output).
5. resolveColors: settings.TuiSettings{} zero value → defaults 212/238; named "pink"/"7" valid → resolved non-default; invalid "#zzz" → falls back to default without error.
6. newTuiModel with custom colors: View() contains SGR escape for the selected row and border (assert via strings.Contains on m.View() of a cursor-on-item render, or render style directly - keep it simple, no exact-screen golden).

## Acceptance
- go test -count=1 ./internal/picker/ ok
- go vet ./internal/... exit 0
- gofmt -l internal/picker/ empty

## Evidence
mkdir -p .mo/evidence; save outputs to .mo/evidence/task-5.txt

## Commit
GIT_MASTER=1 git add -A && GIT_MASTER=1 git commit -m 'test(picker): cover TUI fallback chain and style resolution'
NO PUSH. DoneClaim = commit hash + acceptance summary.
