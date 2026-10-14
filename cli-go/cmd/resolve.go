package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// words are the leading non-flag arguments.
func words(args []string) []string {
	out := []string{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		out = append(out, arg)
	}
	return out
}

// resolution follows the agent-cli/v1 contract shared with the other Reckon CLIs.
type resolution struct {
	Command     string            `json:"command"`
	Effect      string            `json:"effect"` // always read: this CLI has no writes
	Conditional bool              `json:"conditional"`
	Interactive bool              `json:"interactive"`
	Secret      bool              `json:"secret_output"`
	Flags       map[string]string `json:"flags"`
	Args        []string          `json:"args"`
}

// commandsCmd groups offline discovery for agents: resolve answers "what would these
// arguments run?" with the real parser, without credentials or network.
func (a *app) commandsCmd() *cobra.Command {
	c := &cobra.Command{Use: "commands", Short: "Offline command discovery for agents"}
	c.AddCommand(&cobra.Command{Use: "resolve -- <arguments>", Short: "Show the command, effect and flags that arguments would run, without running them",
		DisableFlagParsing: true, Example: "  whatsapp commands resolve -- numbers list",
		RunE: func(cmd *cobra.Command, args []string) error {
			for i, arg := range args {
				if arg == "--" {
					args = args[i+1:]
					break
				}
			}
			a.format = "json"
			probe := NewRoot(strings.NewReader(""), io.Discard, io.Discard)
			// Help, version and completion only print text: reads that touch nothing.
			for _, arg := range args {
				if arg == "--help" || arg == "-h" || arg == "--version" {
					return a.print(resolution{Command: strings.TrimSpace("whatsapp " + strings.Join(words(args), " ")), Effect: "read", Flags: map[string]string{}, Args: []string{}})
				}
			}
			if len(args) > 0 && (args[0] == "help" || args[0] == "completion") {
				return a.print(resolution{Command: "whatsapp " + args[0], Effect: "read", Flags: map[string]string{}, Args: []string{}})
			}
			target, rest, err := probe.Find(args)
			if err != nil || target == probe || !target.Runnable() {
				return fmt.Errorf("arguments do not name a runnable whatsapp command")
			}
			if err := target.ParseFlags(rest); err != nil {
				return err
			}
			if err := target.ValidateArgs(target.Flags().Args()); err != nil {
				return err
			}
			res := resolution{Command: target.CommandPath(), Effect: "read", Flags: map[string]string{}, Args: target.Flags().Args()}
			target.Flags().Visit(func(f *pflag.Flag) { res.Flags[f.Name] = f.Value.String() })
			return a.print(res)
		}})
	return c
}
