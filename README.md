# kubie (Go port)

`kubie` is a Go port of [kubie](https://github.com/kubie-org/kubie), a more powerful
alternative to `kubectx` and `kubens`. It switches Kubernetes contexts and namespaces
in a shell-independent way, and adds commands to run one-off commands against a
context (`exec`), inspect kubeconfig files (`lint`), print the active context
(`info`), edit or delete contexts, export an isolated kubeconfig, and update the
binary itself.

## Install

```sh
go install github.com/khanghua1505/kubie@latest
```

The binary lands in `$GOBIN` (default `$(go env GOPATH)/bin`).

## Usage

Selectable menus are available using `kubie ctx` and `kubie ns`.

`fzf` is an optional dependency. It enables interactive fuzzy selection; without
it, kubie falls back to plain terminal lists.

* `kubie ctx` display selectable menu of contexts
* `kubie ctx <context>` switch current shell to given context (spawns shell if not kubie shell)
* `kubie ctx -` switch back to previous context
* `kubie ctx <context> -r` spawn recursive shell in given context
* `kubie ctx <context> -n <namespace>` spawn a shell in given context and namespace
* `kubie ctx <context> -f <kubeconfig>` load contexts from the given kubeconfig files instead of the installed ones
* `kubie ns` display selectable menu of namespaces
* `kubie ns <namespace>` switch current shell to given namespace
* `kubie ns -` switch back to previous namespace
* `kubie ns <namespace> -r` spawn recursive shell in given namespace
* `kubie ns -u` unset the namespace in the currently active context
* `kubie exec <context> <namespace> -- <cmd> <args>...` execute command in given context and namespace
* `kubie exec <wildcard> <namespace> -- <cmd> <args>...` execute command in all contexts matched by wildcard and in given namespace
* `kubie exec <wildcard> <namespace> -e -- <cmd> <args>...` same, but fail early if any executed command returns a non-zero exit code
* `kubie exec --context-headers <auto|always|never> ...` override the `behavior.print_context_in_exec` setting
* `kubie export <context> <namespace>` print path of the isolated config file for context and namespace
* `kubie info ctx` print name of current context
* `kubie info ns` print name of current namespace
* `kubie info depth` print depth of recursive contexts
* `kubie lint` lint kubeconfig files and report issues
* `kubie edit` display selectable menu of contexts to edit
* `kubie edit <context>` edit the file containing this context
* `kubie edit-config` edit kubie's own config file
* `kubie delete <context>` delete a context, its cluster and its user
* `kubie update` check for a new release and update the local installation
* `kubie generate-completion [SHELL]` generate a completion script for the given shell

## Completion

Generate completion scripts for supported shells:

```sh
source <(kubie generate-completion bash)
source <(kubie generate-completion zsh)
kubie generate-completion fish | source
```

### Settings

Kubie reads its own settings from `~/.kube/kubie.yaml`. The file uses the same keys
as upstream kubie (`shell`, `default_editor`, `configs`, `prompt`, `behavior`,
`hooks`, `fzf`).

## Divergences from upstream

Intentional/known behavioral differences vs kubie-org/kubie (Rust):

- **State file location.** Upstream fixes the state path at the platform data dir (`kubie/state.json`). This port honors `$KUBIE_STATE` when set and falls back to the same platform data dir otherwise.
- **`ctx -n ""`.** Upstream treats an explicit empty namespace as an override. Here an empty `-n` value means "not requested" and falls back to the remembered namespace.
- **`KUBIE_KUBECONFIG` error text.** Upstream: `KUBIE_KUBECONFIG variable not set`. Here: `KUBIE_KUBECONFIG env variable missing`.
- **Namespace-list parsing.** Upstream splits `kubectl` output lines on the first space only; here on the first space or tab. Equivalent for real `kubectl` output.
- **State writes.** Upstream writes the state file in place; here a temp file is written next to it and renamed, so readers never see a partial file. The state directory is also pre-checked and a non-directory path reported explicitly.

## Attribution

Derived from kubie by Simon Bernier St-Pierre (github.com/kubie-org/kubie), Zlib license.
