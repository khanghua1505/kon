package shell

import (
	"os"
	"os/exec"
	"strings"
)

// xonshRCBase is the part of the xonsh rc file that loads the user startup
// files and installs the hook which restores KUBECONFIG before every command.
const xonshRCBase = `
# https://xon.sh/xonshrc.html
from pathlib import Path

files = [
    "/etc/xonshrc",
    "~/.xonshrc",
    "~/.config/xonsh/rc.xsh",
]
for file in files:
    if Path(file).is_file():
        source @(file)
if Path("~/.config/xonsh/rc.d").is_dir():
    for file in path.glob('*.xsh'):
        source @(file)

@events.on_precommand
def __kubie_cmd_pre_exec__(cmd):
    $KUBECONFIG = $KUBIE_KUBECONFIG
`

// xonshRCPrompt is the part of the xonsh rc file that installs the kubie
// prompt.
const xonshRCPrompt = `
$KUBIE_PROMPT='%%PROMPT%%'
import re

# Fanciful prompt-command replacement as xonsh forces the use of PROMPT_FIELDS
for match in re.finditer(r'\$\(([^)]*)\)', $KUBIE_PROMPT):
    command = match.group(1)
    name = command.split().pop()
    $PROMPT_FIELDS[name] = evalx(f'lambda: $({command}).strip()')
    $KUBIE_PROMPT = $KUBIE_PROMPT.replace(f'$({command})', '{' + name + '}')

if $KUBIE_XONSH_USE_RIGHT_PROMPT == "1":
    $RIGHT_PROMPT = $KUBIE_PROMPT + $RIGHT_PROMPT
else:
    $PROMPT = $KUBIE_PROMPT + $PROMPT

del $KUBIE_PROMPT
`

// xonshRCContent returns the content of the rc file handed to xonsh.
func xonshRCContent(info *spawnInfo) string {
	content := xonshRCBase
	if !info.settings.Prompt.Disable {
		content += strings.ReplaceAll(xonshRCPrompt, promptPlaceholder, info.prompt)
	}

	return content + info.settings.Hooks.StartCtx
}

// spawnXonsh runs xonsh with a temporary rc file and returns its exit code.
func spawnXonsh(info *spawnInfo) (int, error) {
	rcPath, err := writeTempFile("kubie-xonshrc", ".xsh", xonshRCContent(info))
	if err != nil {
		return 0, err
	}
	defer os.Remove(rcPath)

	cmd := exec.Command("xonsh", "--rc", rcPath)
	info.envVars.Apply(cmd)

	return run(cmd)
}
