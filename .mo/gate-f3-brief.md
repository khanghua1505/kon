# F3: Real manual QA (branch tui-default-drop-fzf)
Read-only w.r.t. the repo. Work in /Users/khanghua/Workspace/github.com/khanghua1505/kubie (branch tui-default-drop-fzf). Build artifacts to /tmp only. NEVER commit/push.

Build: `go build -o /tmp/kon-qa ./cmd/kon` (check module path first: go list ./cmd/... to find the kon main package).

Fixtures: create /tmp/qa-kubeconfig with 3 fake contexts (dev-a, dev-b, prod-eu clusters/users). export KUBECONFIG=/tmp/qa-kubeconfig. Create /tmp/qa-kon-home; export HOME=/tmp/qa-kon-home for config isolation (kon config at ~/.config/kon/kon.yaml — verify actual path from internal/settings source).

Scenarios (run each, record command + observed output):
1. `kon ctx` with stdin NOT a TTY (pipe </dev/null) and stdout NOT a TTY → expect sorted plain list (listItems branch), exit 0.
2. `kon ctx` piped stdin, stdout NOT TTY, only ONE context in a second kubeconfig → expect immediate selection (single-item branch), context printed/spawned — spawning needs a shell; if it errors on shell spawn, that error text is acceptable evidence of branch taken.
3. TUI in real PTY: `script -q /dev/null /tmp/kon-qa ctx < /dev/null` — TUI starts, EOF/esc path; expect exit 0 cancelled or visible frame captured. If `script` unavailable, use `expect -c`. Record what happened; a clean cancel proves the TUI ran.
4. Config scalar form: kon.yaml with `tui: true` then `tui: false` → with false + non-TTY stdout behavior identical; no load error either way.
5. Config map form: `tui:\n  enabled: true\n  selected_color: pink\n  border_color: "#ff00ff"` → load OK (no error), `kon ctx` still works; bad color `selected_color: "#zzz"` → still no load error (fallback default).
6. `kon ns` and `kon delete ctx` smoke: run with KUBECONFIG fixture, non-TTY → no panic, sane output.
7. `kon ctx` with zero contexts (empty kubeconfig file) → clean error message, exit 1.

Output verdict FIRST LINE: `APPROVE` or `REJECT: <failing scenarios>`. Then per-scenario: command, expected, observed. Do not fix anything.
