# F2: Code quality review (branch tui-default-drop-fzf)
Read-only reviewer. Work in /Users/khanghua/Workspace/github.com/khanghua1505/kubie (branch tui-default-drop-fzf). NEVER edit/commit/push.

Diff range: `git diff main...tui-default-drop-fzf` (ignore .mo/ artifacts).

Review the full diff of product code (cmd/, internal/) for:
1. Dead code, unused exports, leftover fzf/Mode references.
2. Error handling: swallowed errors, missing wraps at boundaries.
3. Duplicated logic that existing repo functions already cover (read the surrounding module first).
4. Naming: constants named for kon primary (Tui, resolveColors), no Mode remnants.
5. Test quality: no sleeps/timing dependencies (grep for time.Sleep in diff), deterministic seams (isTTY override, os.Pipe), each test can fail.
6. go vet ./... and gofmt -l on cmd/ internal/ — run both, must be clean.
7. Config: internal/settings decode must not error on unresolvable colors (fallback 212/238 per user requirement), and TuiSettings dual-form (bool + map) works.
8. Comments: Go doc comments on exported/new funcs only; no noise comments.

Output verdict FIRST LINE: `APPROVE` or `REJECT: <numbered blockers with file:line>`. Only name things that must change (correctness, dead code, test gaps); no style nits that gofmt/vet already gate. Do not fix anything.
