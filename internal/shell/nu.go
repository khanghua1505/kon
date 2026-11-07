package shell

import (
	"os/exec"
	"strings"
)

// spawnNu runs nu with the kubie environment and returns its exit code.
//
// nu has no rc file mechanism, so the environment variables and the prompt
// are set on the command line.
func spawnNu(info *spawnInfo) (int, error) {
	var args strings.Builder

	for _, entry := range info.envVars.entries {
		args.WriteString("$env." + entry.name + " = '" + entry.value + "';")

		if entry.name == "KUBIE_PROMPT_DISABLE" && entry.value == "0" {
			args.WriteString(nuPromptCommand(info.prompt))
		}
	}

	cmd := exec.Command("nu", "-e", args.String())

	return run(cmd)
}

// nuPromptCommand returns the nu command which installs the kubie prompt.
func nuPromptCommand(prompt string) string {
	prompt = strings.ReplaceAll(prompt, `\[\e[31m\]`, "")
	prompt = strings.ReplaceAll(prompt, `\[\e[32m\]`, "")
	prompt = strings.ReplaceAll(prompt, `\[\e[0m\]`, "")
	prompt = strings.ReplaceAll(prompt, "$", "")

	return `$env.PROMPT_COMMAND = { || $"` + prompt + `\n(create_left_prompt)" };`
}
