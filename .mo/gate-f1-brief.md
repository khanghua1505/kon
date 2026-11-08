# F1: Plan compliance audit (branch tui-default-drop-fzf)
You are a read-only auditor. Work in /Users/khanghua/Workspace/github.com/khanghua1505/kubie (branch tui-default-drop-fzf, HEAD should be the row-4 cherry-pick or later). NEVER edit files, NEVER commit, NEVER push.

Plan: .omo/plans/tui-default-drop-fzf.md. Ledger: .mo/ulw-execute/ledger.jsonl.

Audit every row 1-7 against the plan's per-row Acceptance criteria. For each row:
1. Run its acceptance command(s) yourself (go build/test/vet, greps). Redirect long output to /tmp and read back.
2. Check the row's evidence file exists under .mo/evidence/ and matches (task-1..task-7 naming per plan; task-3/4 also exist in lane worktrees but main evidence counts).
3. Grep the whole repo for fzf: `grep -rni fzf --include='*.go' . ; grep -in fzf README.md` → both must be 0 (vendor/ excluded).
4. Mode enum: `grep -rn 'ModeAuto\|ModeFzf\|ModeTerminal\|ModeTui' --include='*.go' .` → 0 matches repo-wide.
5. Config dual-form: table tests in internal/settings cover scalar true/false, map partial, invalid color → default (no error), stale fzf key ignored.
6. Fallback chain in internal/picker/picker.go matches plan Must-have order exactly: single → TUI (enabled && both TTY) → promptList (Out TTY) → listItems.
7. One commit per row on branch: `git log --oneline main..tui-default-drop-fzf` - list commits, map each to its row, flag extra/missing/mis-scoped commits. Note: 9653b4f was a user-authored commit of staged cmd fixes (accepted); cherry-pick hashes differ from lane hashes (accepted).

Output verdict FIRST LINE: `APPROVE` or `REJECT: <blockers>`. Then per-row evidence table (row, acceptance command, result). REJECT only for real compliance gaps, not style. Do not fix anything.
