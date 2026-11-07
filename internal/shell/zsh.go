package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// zshRCBase is the part of the zsh rc file that loads the user startup files
// and installs the hook which restores KUBECONFIG before every command.
const zshRCBase = `
# Reference for loading behavior
# https://shreevatsa.wordpress.com/2008/03/30/zshbash-startup-files-loading-order-bashrc-zshrc-etc/

if [[ -f "/etc/zshenv" ]] ; then
    source "/etc/zshenv"
elif [[ -f "/etc/zsh/zshenv" ]] ; then
    source "/etc/zsh/zshenv"
fi

if [[ -f "$HOME/.zshenv" ]] ; then
    tmp_ZDOTDIR=$ZDOTDIR
    source "$HOME/.zshenv"
    # If the user has overridden $ZDOTDIR, we save that in $_KUBIE_USER_ZDOTDIR for later reference
    # and reset $ZDOTDIR
    if [[ "$tmp_ZDOTDIR" != "$ZDOTDIR" ]]; then
        _KUBIE_USER_ZDOTDIR=$ZDOTDIR
        ZDOTDIR=$tmp_ZDOTDIR
        unset tmp_ZDOTDIR
    fi
fi

# If a zsh_history file exists, copy it over before zsh initialization so history is maintained
if [[ -f "$HOME/.zsh_history" ]] ; then
    cp $HOME/.zsh_history $ZDOTDIR
fi

KUBIE_LOGIN_SHELL=0
if [[ "$OSTYPE" == "darwin"* ]] ; then
    KUBIE_LOGIN_SHELL=1
fi

if [[ -f "/etc/zprofile" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "/etc/zprofile"
elif [[ -f "/etc/zsh/zprofile" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "/etc/zsh/zprofile"
fi

if [[ -f "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zprofile" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zprofile"
fi

if [[ -f "/etc/zshrc" ]] ; then
    source "/etc/zshrc"
elif [[ -f "/etc/zsh/zshrc" ]] ; then
    source "/etc/zsh/zshrc"
fi

if [[ -f "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zshrc" ]] ; then
    ZDOTDIR=$_KUBIE_USER_ZDOTDIR \
        source "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zshrc"
fi

if [[ -f "/etc/zlogin" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "/etc/zlogin"
elif [[ -f "/etc/zsh/zlogin" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "/etc/zsh/zlogin"
fi

if [[ -f "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zlogin" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zlogin"
fi

unset _KUBIE_USER_ZDOTDIR

autoload -Uz add-zsh-hook

# This function sets the proper KUBECONFIG variable before a command runs,
# in case something overwrote it.
function __kubie_cmd_pre_exec__() {
    export KUBECONFIG="$KUBIE_KUBECONFIG"
}

add-zsh-hook preexec __kubie_cmd_pre_exec__
`

// zshRCPrompt is the part of the zsh rc file that installs the kubie prompt.
const zshRCPrompt = `
# Activate prompt substitution.
setopt PROMPT_SUBST

# This function fixes the prompt via a precmd hook.
function __kubie_cmd_pre_cmd__() {
    local KUBIE_PROMPT=$'%%PROMPT%%'

    # If KUBIE_ZSH_USE_RPS1 is set, we use RPS1 instead of PS1.
    if [[ "$KUBIE_ZSH_USE_RPS1" == "1" ]] ; then

        # Avoid modifying RPS1 again if the RPS1 has not been reset.
        if [[ "$RPS1" != *"$KUBIE_PROMPT"* ]] ; then

            # If RPS1 is empty, we do not seperate with a space.
            if [[ -z "$RPS1" ]] ; then
                RPS1="$KUBIE_PROMPT"
            else
                RPS1="$KUBIE_PROMPT $RPS1"
            fi
        fi
    else
        # Avoid modifying PS1 again if the PS1 has not been reset.
        if [[ "$PS1" != *"$KUBIE_PROMPT"* ]] ; then
            PS1="$KUBIE_PROMPT $PS1"
        fi
    fi
}

# When promptinit is activated, a precmd hook which updates PS1 is installed.
# In order to inject the kubie PS1 when promptinit is activated, we must
# also add our own precmd hook which modifies PS1 after promptinit themes.
add-zsh-hook precmd __kubie_cmd_pre_cmd__
`

// zshRCContent returns the content of the .zshrc file handed to zsh.
func zshRCContent(info *spawnInfo) string {
	content := zshRCBase
	if !info.settings.Prompt.Disable {
		content += strings.ReplaceAll(zshRCPrompt, promptPlaceholder, info.prompt)
	}

	return content + info.settings.Hooks.StartCtx
}

// spawnZsh runs zsh with a temporary ZDOTDIR holding the kubie .zshrc file
// and returns its exit code.
func spawnZsh(info *spawnInfo) (int, error) {
	dir, err := writeTempDir("kubie-zshrc")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(dir)

	rcPath := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(zshRCContent(info)), 0o600); err != nil {
		return 0, err
	}

	cmd := exec.Command("zsh")
	cmd.Env = append(os.Environ(), "ZDOTDIR="+dir)
	info.envVars.Apply(cmd)

	code, err := run(cmd)
	if err != nil {
		return code, err
	}

	if info.settings.Hooks.StopCtx != "" {
		hookPath, err := writeTempFile("kubie-zsh-exit-hook", ".zsh", info.settings.Hooks.StopCtx)
		if err != nil {
			return code, err
		}
		defer os.Remove(hookPath)

		exitCmd := exec.Command("zsh", hookPath)
		info.envVars.Apply(exitCmd)

		if _, err := run(exitCmd); err != nil {
			return code, err
		}
	}

	return code, nil
}
