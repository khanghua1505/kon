# kon (Go port of kubie)

<p align="center">
  <a href="https://www.zerochan.net/1115386">
    <img src="https://s3.zerochan.net/Kon.%28BLEACH%29.240.1115386.avif" alt="Kon (BLEACH)" width="240" />
  </a>
</p>

`kon` is a Go port of [kubie](https://github.com/kubie-org/kubie), a more powerful
alternative to `kubectx` and `kubens`. It switches Kubernetes contexts and namespaces
in a shell-independent way, and adds commands to run one-off commands against a
context (`exec`), inspect kubeconfig files (`lint`), print the active context
(`info`), edit or delete contexts, export an isolated kubeconfig, and update the
binary itself.

## Install

```sh
go install github.com/khanghua1505/kon/cmd/kon@latest
```

The binary lands in `$GOBIN` (default `$(go env GOPATH)/bin`).

## Usage

Selectable menus are available using `kon ctx` and `kon ns`.

`kon` ships a built-in TUI picker powered by Bubble Tea, enabled by default.
Picker fallback order is: select a single item directly, open the TUI when
enabled on a terminal, show a terminal list prompt, then print a numbered list
for non-interactive output. Set `tui.enabled: false` (or scalar `tui: false`)
to skip the TUI and use the terminal list.

* `kon ctx` display selectable menu of contexts
* `kon ctx <context>` switch current shell to given context (spawns shell if not kon shell)
* `kon ctx -` switch back to previous context
* `kon ctx <context> -r` spawn recursive shell in given context
* `kon ctx <context> -n <namespace>` spawn a shell in given context and namespace
* `kon ctx <context> -f <kubeconfig>` load contexts from the given kubeconfig files instead of the installed ones
* `kon ns` display selectable menu of namespaces
* `kon ns <namespace>` switch current shell to given namespace
* `kon ns -` switch back to previous namespace
* `kon ns <namespace> -r` spawn recursive shell in given namespace
* `kon ns -u` unset the namespace in the currently active context
* `kon exec <context> <namespace> -- <cmd> <args>...` execute command in given context and namespace
* `kon exec <wildcard> <namespace> -- <cmd> <args>...` execute command in all contexts matched by wildcard and in given namespace
* `kon exec <wildcard> <namespace> -e -- <cmd> <args>...` same, but fail early if any executed command returns a non-zero exit code
* `kon exec --context-headers <auto|always|never> ...` override the `behavior.print_context_in_exec` setting
* `kon export <context> <namespace>` print path of the isolated config file for context and namespace
* `kon info ctx` print name of current context
* `kon info ns` print name of current namespace
* `kon info depth` print depth of recursive contexts
* `kon lint` lint kubeconfig files and report issues
* `kon edit` display selectable menu of contexts to edit
* `kon edit <context>` edit the file containing this context
* `kon edit-config` edit kon's own config file
* `kon delete <context>` delete a context, its cluster and its user
* `kon update` check for a new release and update the local installation
* `kon generate-completion [SHELL]` generate a completion script for the given shell

## Completion

Generate completion scripts for supported shells:

```sh
source <(kon generate-completion bash)
source <(kon generate-completion zsh)
kon generate-completion fish | source
```

### Settings

kon reads its own settings from `~/.kube/kon.yaml`. If that file does not
exist but `~/.kube/kubie.yaml` does, the kubie file is used instead, so
existing setups keep working. The file accepts these keys:
`shell`, `default_editor`, `tui`, `configs`, `prompt`, `behavior`, `hooks`.


```yaml
# Force kubie to use a particular shell, if unset detect shell currently in use.
# Possible values: bash, dash, fish, xonsh, zsh
# Default: unset
shell: bash

# For the commands `kubie edit/edit-config`
# Possible values: Any installed text editor
# Default: unset
default_editor: vim

# Configure the built-in TUI picker. Scalar `tui: true` and `tui: false` are
# also accepted. Set `enabled: false` to fall back to the terminal list.
# Colors accept 16 ANSI names (black, red, green, yellow, blue, magenta,
# cyan, white, brightblack, brightred, brightgreen, brightyellow, brightblue,
# brightmagenta, brightcyan, brightwhite), a decimal ANSI index such as 212,
# or a hex value such as "#ff5f87".
# Invalid colors fall back to 212 (selected) and 238 (border) without error.
# Legacy picker keys are ignored silently.
# Default: enabled
tui:
    enabled: true
    selected_color: magenta
    border_color: "#ff5f87"

# Configure where to look for kubernetes config files.
configs:

    # Include these globs.
    # Default: values listed below.
    include:
        - ~/.kube/config
        - ~/.kube/*.yml
        - ~/.kube/*.yaml
        - ~/.kube/configs/*.yml
        - ~/.kube/configs/*.yaml
        - ~/.kube/kubie/*.yml
        - ~/.kube/kubie/*.yaml

    # Exclude these globs.
    # Default: values listed below.
    # Note: kubie's own config file is always excluded.
    exclude:
        - ~/.kube/kubie.yaml

# Prompt settings.
prompt:
    # Disable kubie's custom prompt inside of a kubie shell. This is useful
    # when you already have a prompt displaying kubernetes information.
    # Default: false
    disable: false

    # When using recursive contexts, show depth when larger than 1.
    # Default: true
    show_depth: true

    # When using zsh, show context and namespace on the right-hand side using RPS1.
    # Default: false
    zsh_use_rps1: false

    # When using fish, show context and namespace on the right-hand side.
    # Default: false
    fish_use_rprompt: false

    # When using xonsh, show context and namespace on the right-hand side.
    # Default: false
    xonsh_use_right_prompt: false

# Behavior
behavior:
    # Namespace validation and switching behavior.  Set to "false" if you do not have
    # the right to list namespaces.
    # Valid values:
    #   true:    Make sure the namespace exists with `kubectl get namespaces`.
    #   false:   Switch namespaces without validation.
    #   partial: Check for partial matches when running `kubie ns <namespace>`
    #            and no exact match is found:
    #              - if exactly one namespace partially matches, switch to that namespace
    #              - if multiple namespaces partially match, select from those
    # Default: true
    validate_namespaces: true

    # Enable or disable the printing of the 'CONTEXT => ...' headers when running
    # `kubie exec`.
    # Valid values:
    #   auto:   Prints context headers only if stdout is a TTY. Piping/redirecting
    #           kubie output will auto-disable context headers.
    #   always: Always prints context headers, even if stdout is not a TTY.
    #   never:  Never prints context headers.
    # Default: auto
    print_context_in_exec: auto

    # Parse the CONTEXT_NAME argument to `kubie exec` and `kubie export` as a
    # space-delimited list, allowing multiple patterns to be specified.
    # Example:
    #   kubie exec 'dev-* pre-* staging-1 prod-2' kube-system -- kubectl get po
    # Default: false
    allow_multiple_context_patterns: false

# Optional start and stop hooks
hooks:
    # A command hook to run when a CTX is started.  
    # This example re-labels your terminal window
    # Default: none
    start_ctx: >
        echo -en "\033]1; `kubie info ctx`|`kubie info ns` \007"

    # A command hook to run when a CTX is stopped
    # This example sets the terminal back to the shell name
    # Default: none
    stop_ctx: >
        echo -en "\033]1; $SHELL \007"
```

### Environment variables

Inside a kon shell, `KON_*` variables describe the active shell:
`KON_ACTIVE`, `KON_DEPTH`, `KON_KUBECONFIG` (also written as
`KON_KUBE_CONFIG`), `KON_SESSION`, `KON_STATE`, plus per-shell prompt flags.
Commands read `KON_*` first and fall back to the legacy `KUBIE_*` variables,
so shells spawned by the original kubie keep working. kon-spawned shells
export both families with the same values, which keeps kubie-era prompt
scripts and `KUBIE_*`-aware tooling functional.

## Divergences from upstream

Intentional/known behavioral differences vs kubie-org/kubie (Rust):

- **State file location.** Upstream fixes the state path at the platform data dir (`kubie/state.json`). This port honors `$KON_STATE` when set and falls back to the same platform data dir otherwise. (`$KUBIE_STATE` is read as a legacy fallback.)
- **`ctx -n ""`.** Upstream treats an explicit empty namespace as an override. Here an empty `-n` value means "not requested" and falls back to the remembered namespace.
- **`KON_KUBECONFIG` error text.** Upstream: `KON_KUBECONFIG variable not set`. Here: `KON_KUBECONFIG env variable missing`.
- **Namespace-list parsing.** Upstream splits `kubectl` output lines on the first space only; here on the first space or tab. Equivalent for real `kubectl` output.
- **State writes.** Upstream writes the state file in place; here a temp file is written next to it and renamed, so readers never see a partial file. The state directory is also pre-checked and a non-directory path reported explicitly.

## Attribution

Derived from kubie by Simon Bernier St-Pierre (github.com/kubie-org/kubie), Zlib license.
