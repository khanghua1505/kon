package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// aliasPart is one segment of a generated alias: the token it contributes to
// the alias name, the command fragment it expands to, the alias tokens that
// must accompany it (oneof) and the ones it cannot be combined with (incompat).
type aliasPart struct {
	alias    string
	full     string
	oneof    []string
	incompat []string
}

// aliasPartGroup holds the alias parts that occupy one slot of the alias
// name. takeOne limits the slot to a single part, optional allows the slot
// to be empty.
type aliasPartGroup struct {
	items    []aliasPart
	optional bool
	takeOne  bool
}

// newKubectlAliasCmd builds the kubectl-alias command.
func newKubectlAliasCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "kubectl-alias [SHELL]",
		Short:     "Generate kubectl aliases, in the style of github.com/ahmetb/kubectl-aliases.",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "nushell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ""
			if len(args) == 1 {
				shell = args[0]
			}
			return generateKubectlAliases(cmd, shell)
		},
	}
}

// generateKubectlAliases writes the alias definitions of the given shell.
func generateKubectlAliases(cmd *cobra.Command, shell string) error {
	if shell == "" {
		if detected, ok := detectShell(os.Getenv("SHELL")); ok {
			switch detected {
			case "bash", "zsh", "fish", "nushell":
				shell = detected
			}
		}
	}

	var format string
	switch shell {
	case "", "bash", "zsh":
		format = "alias %s='%s'"
	case "fish":
		format = "abbr --add %s \"%s\""
	case "nushell":
		format = "alias %s = %s"
	default:
		return fmt.Errorf("unsupported shell %q, supported shells are bash, zsh, fish and nushell", shell)
	}

	out := cmd.OutOrStdout()

	seen := make(map[string]bool)
	for _, parts := range aliasCombinations(kubectlAliasGroups()) {
		alias := make([]string, len(parts))
		full := make([]string, len(parts))
		for i, part := range parts {
			alias[i] = part.alias
			full[i] = part.full
		}
		name := strings.Join(alias, "")
		if seen[name] {
			fmt.Fprintf(cmd.ErrOrStderr(), "Alias conflict detected: %s\n", name)
		}
		seen[name] = true

		if _, err := fmt.Fprintf(out, format+"\n", name, strings.Join(full, " ")); err != nil {
			return err
		}
	}

	return nil
}

// kubectlAliasGroups defines the alias segments, ported from the
// generate_aliases.py tables of github.com/ahmetb/kubectl-aliases.
func kubectlAliasGroups() []aliasPartGroup {
	cmds := []aliasPart{
		{alias: "k", full: "kubectl"},
	}
	globs := []aliasPart{
		{alias: "sys", full: "--namespace=kube-system"},
	}
	ops := []aliasPart{
		{alias: "a", full: "apply --recursive -f"},
		{alias: "ak", full: "apply -k", incompat: []string{"sys"}},
		{alias: "k", full: "kustomize", incompat: []string{"sys"}},
		{alias: "ex", full: "exec -i -t"},
		{alias: "lo", full: "logs -f"},
		{alias: "lop", full: "logs -f -p"},
		{alias: "p", full: "proxy", incompat: []string{"sys"}},
		{alias: "pf", full: "port-forward", incompat: []string{"sys"}},
		{alias: "g", full: "get"},
		{alias: "d", full: "describe"},
		{alias: "rm", full: "delete"},
		{alias: "run", full: "run --rm --restart=Never --image-pull-policy=IfNotPresent -i -t"},
	}
	res := []aliasPart{
		{alias: "po", full: "pods", oneof: []string{"g", "d", "rm"}},
		{alias: "dep", full: "deployment", oneof: []string{"g", "d", "rm"}},
		{alias: "sts", full: "statefulset", oneof: []string{"g", "d", "rm"}},
		{alias: "svc", full: "service", oneof: []string{"g", "d", "rm"}},
		{alias: "ing", full: "ingress", oneof: []string{"g", "d", "rm"}},
		{alias: "cm", full: "configmap", oneof: []string{"g", "d", "rm"}},
		{alias: "sec", full: "secret", oneof: []string{"g", "d", "rm"}},
		{alias: "no", full: "nodes", oneof: []string{"g", "d"}, incompat: []string{"sys"}},
		{alias: "ns", full: "namespaces", oneof: []string{"g", "d", "rm"}, incompat: []string{"sys"}},
	}
	args := []aliasPart{
		{alias: "oyaml", full: "-o=yaml", oneof: []string{"g"}, incompat: []string{"owide", "ojson", "sl"}},
		{alias: "owide", full: "-o=wide", oneof: []string{"g"}, incompat: []string{"oyaml", "ojson"}},
		{alias: "ojson", full: "-o=json", oneof: []string{"g"}, incompat: []string{"owide", "oyaml", "sl"}},
		{alias: "all", full: "--all-namespaces", oneof: []string{"g", "d"}, incompat: []string{"rm", "f", "no", "ns", "sys"}},
		{alias: "sl", full: "--show-labels", oneof: []string{"g"}, incompat: []string{"oyaml", "ojson"}},
		{alias: "all", full: "--all", oneof: []string{"rm"}},
		{alias: "w", full: "--watch", oneof: []string{"g"}, incompat: []string{"oyaml", "ojson", "owide"}},
	}
	resTypes := []string{"po", "dep", "sts", "svc", "ing", "cm", "sec", "no", "ns"}
	positional := []aliasPart{
		{alias: "f", full: "--recursive -f", oneof: []string{"g", "d", "rm"}, incompat: append(append([]string{}, resTypes...), "all", "l", "sys")},
		{alias: "l", full: "-l", oneof: []string{"g", "d", "rm"}, incompat: []string{"f", "all"}},
		{alias: "n", full: "--namespace", oneof: []string{"g", "d", "rm", "lo", "ex", "pf"}, incompat: []string{"ns", "no", "sys", "all"}},
	}

	return []aliasPartGroup{
		{items: cmds},
		{items: globs, optional: true},
		{items: ops, optional: true, takeOne: true},
		{items: res, optional: true, takeOne: true},
		{items: args, optional: true},
		{items: positional, optional: true, takeOne: true},
	}
}

// aliasCombinations expands the part groups into every valid command, in the
// order itertools.combinations and itertools.permutations yield them.
func aliasCombinations(groups []aliasPartGroup) [][]aliasPart {
	out := [][]aliasPart{{}}
	for _, group := range groups {
		n := len(group.items)
		if group.takeOne {
			n = 1
		}

		var combos [][]aliasPart
		for j := 0; j <= n; j++ {
			if j == 0 && !group.optional {
				continue
			}
			eachCombination(group.items, j, func(c []aliasPart) {
				if aliasPartsCompatible(c) {
					combos = append(combos, c)
				}
			})
		}

		if group.optional {
			var permuted [][]aliasPart
			for _, combo := range combos {
				eachPermutation(combo, func(p []aliasPart) {
					permuted = append(permuted, p)
				})
			}
			combos = permuted
		}

		var next [][]aliasPart
		for _, segment := range combos {
			for _, prefix := range out {
				candidate := make([]aliasPart, 0, len(prefix)+len(segment))
				candidate = append(candidate, prefix...)
				candidate = append(candidate, segment...)
				if aliasPartsValid(candidate) {
					next = append(next, candidate)
				}
			}
		}
		out = next
	}
	return out
}

// eachCombination calls fn for every j-element combination of items, in
// lexicographic index order.
func eachCombination(items []aliasPart, j int, fn func([]aliasPart)) {
	current := make([]aliasPart, 0, j)
	var walk func(start int)
	walk = func(start int) {
		if len(current) == j {
			fn(append([]aliasPart{}, current...))
			return
		}
		need := j - len(current)
		for i := start; i <= len(items)-need; i++ {
			current = append(current, items[i])
			walk(i + 1)
			current = current[:len(current)-1]
		}
	}
	walk(0)
}

// eachPermutation calls fn for every ordering of items, in lexicographic
// index order, matching itertools.permutations.
func eachPermutation(items []aliasPart, fn func([]aliasPart)) {
	used := make([]bool, len(items))
	current := make([]aliasPart, 0, len(items))
	var walk func()
	walk = func() {
		if len(current) == len(items) {
			fn(append([]aliasPart{}, current...))
			return
		}
		for i := range items {
			if used[i] {
				continue
			}
			used[i] = true
			current = append(current, items[i])
			walk()
			current = current[:len(current)-1]
			used[i] = false
		}
	}
	walk()
}

// aliasPartsCompatible reports whether no part conflicts with another.
func aliasPartsCompatible(parts []aliasPart) bool {
	present := aliasPartTokens(parts)
	for _, part := range parts {
		for _, token := range part.incompat {
			if present[token] {
				return false
			}
		}
	}
	return true
}

// aliasPartsValid reports whether every part's requirements hold.
func aliasPartsValid(parts []aliasPart) bool {
	if !aliasPartsCompatible(parts) {
		return false
	}
	present := aliasPartTokens(parts)
	for _, part := range parts {
		if len(part.oneof) == 0 {
			continue
		}
		found := false
		for _, token := range part.oneof {
			if present[token] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func aliasPartTokens(parts []aliasPart) map[string]bool {
	present := make(map[string]bool, len(parts))
	for _, part := range parts {
		present[part.alias] = true
	}
	return present
}
