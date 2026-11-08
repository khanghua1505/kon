# F4: Ideal-state fidelity (branch tui-default-drop-fzf)
Read-only auditor. Work in /Users/khanghua/Workspace/github.com/khanghua1505/kubie (branch tui-default-drop-fzf). NEVER edit/commit/push.

The plan's Ideal State (from .mo/plans/tui-default-drop-fzf.md): fzf fully gone; TUI is the default picker at every selection site; internal/tui/style named-color mapping (16 names) with hex/ANSI fallback to defaults; TuiSettings dual-form config (scalar bool + map form); unresolvable colors silently fall back to defaults (no Load error); whole Mode enum deleted, tui.enabled the sole switch; fallback chain single→TUI→promptList→listItems; README documents TUI default, config forms, fallback, and no longer mentions fzf.

Check delivered behavior against the Ideal State 1:1. For each ideal-state clause: cite file:line + command evidence that the branch delivers it. Any shortfall is a REJECT with the exact clause; per plan footer, shortfalls become new `- [ ] N.` rows, never notes.

Also read README.md fully: does it document (a) TUI default picker + keys, (b) tui config both forms with color names table, (c) fallback behavior when not a terminal, (d) zero fzf mentions. Check upstream-divergence section stays accurate.

Output verdict FIRST LINE: `APPROVE` or `REJECT: <clauses unmet>`. Then clause→evidence table. Do not fix anything.
