# kon

<img src="assets/kon-icon.png" alt="Kon (BLEACH)" title="Kon from BLEACH" align="right" width="240"/>

`kon` is a Go port of [kubie](https://github.com/kubie-org/kubie), an alternative to `kubectx`, `kubens` and the
`k on` prompt modification script. It offers context switching, namespace switching and prompt modification in a way
that makes each shell independent from others. It also has support for split configuration files, meaning it can load
Kubernetes contexts from multiple files. You can configure the paths where kon will look for contexts, see the
[settings](#settings) section.

kon also has other nice features such as `kon exec` which allows you to execute commands in a context and a namespace
without having to spawn a shell, `kon lint` which scans your k8s config files for issues and informs you of what they
are, and a built-in TUI picker that removes kubie's external fuzzy-finder dependency.

- [kon](#kon)
  - [Installation](#installation)
    - [Go](#go)
    - [Autocompletion](#autocompletion)
  - [Usage](#usage)
  - [Settings](#settings)

## Installation

### Go

You can build `kon` from source using `go install`. If you do not have Go installed, go to [go.dev](https://go.dev/dl/)
to get one. Then you can run the command below and the `kon` binary will land in `$GOBIN`
(default `$(go env GOPATH)/bin`).

```sh
go install github.com/khanghua1505/kon/cmd/kon@latest
```

### Autocompletion

Autocompletion can be enabled by adding one of the following lines to your shell's configuration file, e.g. `~/.bashrc`
or `~/.zshrc`:

```sh
# Bash / Zsh / others
source <(kon generate-completion)
# Fish
kon generate-completion fish | source
```

## Usage

Selectable menus are available when using `kon ctx` and `kon ns`. They use the built-in TUI picker powered by
Bubble Tea, enabled by default. Picker fallback order is: select a single item directly, open the TUI when enabled on
a terminal, show a terminal list prompt, then print a numbered list for non-interactive output. Set
`tui.enabled: false` (or scalar `tui: false`) to skip the TUI and use the terminal list.

---

* `kon ctx` display a selectable menu of contexts
* `kon ctx <context>` switch the current shell to the given context (spawns a shell if not a kon shell)
* `kon ctx -` switch back to the previous context (session history inside a kon shell, last globally-used context otherwise)
* `kon ctx <context> -r` spawn a recursive shell in the given context
* `kon ctx <context> -n <namespace>` spawn a shell in the given context and namespace
* `kon ctx <context> -f <kubeconfig>` load contexts from the given kubeconfig files instead of the installed ones
* `kon ns` display a selectable menu of namespaces
* `kon ns <namespace>` switch the current shell to the given namespace
* `kon ns -` switch back to the previous namespace
* `kon ns <namespace> -r` spawn a recursive shell in the given namespace
* `kon ns -u` unset the namespace in the currently active context
* `kon exec <context> <namespace> -- <cmd> <args>...` execute a command in the given context and namespace
* `kon exec <context> -- <cmd> <args>...` execute a command in the given context and the current namespace
* `kon exec -- <cmd> <args>...` execute a command in the current context and namespace (inside a kon shell)
* `kon exec <wildcard> <namespace> -- <cmd> <args>...` execute a command in all the contexts matched by the wildcard and in the given namespace
* `kon exec <wildcard> <namespace> -e -- <cmd> <args>...` same, but fail early if any executed command returns a non-zero exit code
* `kon exec --context-headers <auto|always|never> ...` override the `behavior.print_context_in_exec` setting
* `kon export <context> <namespace>` prints the path to an isolated config file for a context and namespace
* `kon edit` display a selectable menu of contexts to edit
* `kon edit <context>` edit the file that contains this context
* `kon edit-config` edit kon's own config file
* `kon delete <context>` delete a context, its cluster and its user
* `kon info ctx` print name of current context
* `kon info ns` print name of current namespace
* `kon info depth` print depth of recursive contexts
* `kon lint` lint k8s config files for issues
* `kon update` check for a new release and update the local installation

## Settings

You can customize kon's behavior with the `~/.kube/kon.yaml` file. If that file does not exist but `~/.kube/kubie.yaml`
does, the kubie file is used instead, so existing setups keep working. The file accepts these keys: `shell`,
`default_editor`, `tui`, `configs`, `prompt`, `behavior`, `hooks`. The settings available and their defaults are
available below.

```yaml
# Force kon to use a particular shell, if unset detect shell currently in use.
# Possible values: bash, dash, fish, xonsh, zsh
# Default: unset
shell: bash

# For the commands `kon edit/edit-config`
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
    # Disable kubie's custom prompt inside of a kon shell. This is useful
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
    #   partial: Check for partial matches when running `kon ns <namespace>`
    #            and no exact match is found:
    #              - if exactly one namespace partially matches, switch to that namespace
    #              - if multiple namespaces partially match, select from those
    # Default: true
    validate_namespaces: true

    # Enable or disable the printing of the 'CONTEXT => ...' headers when running
    # `kon exec`.
    # Valid values:
    #   auto:   Prints context headers only if stdout is a TTY. Piping/redirecting
    #           kon output will auto-disable context headers.
    #   always: Always prints context headers, even if stdout is not a TTY.
    #   never:  Never prints context headers.
    # Default: auto
    print_context_in_exec: auto

    # Parse the CONTEXT_NAME argument to `kon exec` and `kon export` as a
    # space-delimited list, allowing multiple patterns to be specified.
    # Example:
    #   kon exec 'dev-* pre-* staging-1 prod-2' kube-system -- kubectl get po
    # Default: false
    allow_multiple_context_patterns: false

# Optional start and stop hooks
hooks:
    # A command hook to run when a CTX is started.
    # This example re-labels your terminal window
    # Default: none
    start_ctx: >
        echo -en "\033]1; `kon info ctx`|`kon info ns` \007"

    # A command hook to run when a CTX is stopped
    # This example sets the terminal back to the shell name
    # Default: none
    stop_ctx: >
        echo -en "\033]1; $SHELL \007"
```

