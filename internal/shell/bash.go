package shell

import (
	"os"
	"os/exec"
	"strings"
)

// bashRCBase is the part of the bash rc file that loads the user startup
// files and installs the hook which restores KUBECONFIG before every command.
const bashRCBase = `
KUBIE_LOGIN_SHELL=0
if [[ "$OSTYPE" == "darwin"* ]] ; then
    KUBIE_LOGIN_SHELL=1
fi

# Reference for loading behavior
# https://shreevatsa.wordpress.com/2008/03/30/zshbash-startup-files-loading-order-bashrc-zshrc-etc/


if [[ "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    if [[ -f "/etc/profile" ]] ; then
        source "/etc/profile"
    fi

    if [[ -f "$HOME/.bash_profile" ]] ; then
        source "$HOME/.bash_profile"
    elif [[ -f "$HOME/.bash_login" ]] ; then
        source "$HOME/.bash_login"
    elif [[ -f "$HOME/.profile" ]] ; then
        source "$HOME/.profile"
    fi
else
    if [[ -f "/etc/bash.bashrc" ]] ; then
        source "/etc/bash.bashrc"
    fi

    if [[ -f "$HOME/.bashrc" ]] ; then
        source "$HOME/.bashrc"
    fi
fi

function __kubie_cmd_pre_exec__() {
    export KUBECONFIG="$KUBIE_KUBECONFIG"
}

trap '__kubie_cmd_pre_exec__' DEBUG
`

// bashRCPrompt is the part of the bash rc file that installs the kubie prompt.
const bashRCPrompt = `
KUBIE_PROMPT='%%PROMPT%%'
PS1="$KUBIE_PROMPT $PS1"
unset KUBIE_PROMPT
`

// bashRCContent returns the content of the rc file handed to bash.
func bashRCContent(info *spawnInfo) string {
	content := bashRCBase
	if !info.settings.Prompt.Disable {
		content += strings.ReplaceAll(bashRCPrompt, promptPlaceholder, info.prompt)
	}

	return content + info.settings.Hooks.StartCtx
}

// spawnBash runs bash with a temporary rc file and returns its exit code.
func spawnBash(info *spawnInfo) (int, error) {
	rcPath, err := writeTempFile("kubie-bashrc", ".bash", bashRCContent(info))
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.Remove(rcPath) }()

	cmd := exec.Command("bash", "--rcfile", rcPath)
	info.envVars.Apply(cmd)

	code, err := run(cmd)
	if err != nil {
		return code, err
	}

	// upstream gates the bash exit hook on start_ctx, not stop_ctx
	if info.settings.Hooks.StartCtx != "" {
		hookPath, err := writeTempFile("kubie-bash-exit-hook", ".bash", info.settings.Hooks.StopCtx)
		if err != nil {
			return code, err
		}
		defer func() { _ = os.Remove(hookPath) }()

		exitCmd := exec.Command("bash", hookPath)
		info.envVars.Apply(exitCmd)

		if _, err := run(exitCmd); err != nil {
			return code, err
		}
	}

	return code, nil
}
