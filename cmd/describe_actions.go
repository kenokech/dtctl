package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/dynatrace-oss/dtctl/pkg/output"
	"github.com/dynatrace-oss/dtctl/pkg/resources/appengine"
	"github.com/dynatrace-oss/dtctl/pkg/stability"
)

// describeActionCmd describes an app action
var describeActionCmd = newDescribeActionCmd()

func newDescribeActionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "action <app-id>/<action-name>",
		Aliases: []string{"actions"},
		Short:   "Describe an App Engine action",
		Long: `Show detailed information about an app action.

Actions are declared in an app's manifest. Any key of that entry without a
field of its own is listed under "Additional manifest keys".

Examples:
  # Describe an action
  dtctl describe action dynatrace.automations/execute-dql-query

  # Output as JSON
  dtctl describe action dynatrace.automations/execute-dql-query -o json

  # Output as YAML
  dtctl describe action dynatrace.automations/execute-dql-query -o yaml
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, printer, err := setup(cmdContext(cmd))
			if err != nil {
				return err
			}

			handler := appengine.NewHandler(c)

			action, err := handler.GetAction(args[0])
			if err != nil {
				return err
			}

			// For table output, show detailed human-readable information
			if outputFormat(cmdContext(cmd)) == "table" {
				return describeActionTable(currentStdout(cmdContext(cmd)), action)
			}

			// For other formats, use standard printer
			enrichAgent(printer, "describe", "action")
			return printer.Print(action)
		},
	}
	// Declared stable: the invocation and output contract of this command is
	// additive-only. Stable is never implied -- see AGENTS.md "Stability Tiers".
	stability.MarkStable(c)
	return c
}

// describeActionTable renders the vertical view. Stateful prints only when
// set: the manifest omits it rather than declaring it false, so "Stateful: no"
// would assert something the manifest never said.
func describeActionTable(w io.Writer, action *appengine.AppAction) error {
	const kw = 14

	output.FprintDescribeKV(w, "Action:", kw, "%s", action.ActionName)
	output.FprintDescribeKV(w, "Full Name:", kw, "%s", action.FullName)
	output.FprintDescribeKV(w, "App:", kw, "%s (%s)", action.AppName, action.AppID)
	if action.Title != "" {
		output.FprintDescribeKV(w, "Title:", kw, "%s", action.Title)
	}
	if action.Description != "" {
		output.FprintDescribeKV(w, "Description:", kw, "%s", action.Description)
	}
	if action.Stateful {
		output.FprintDescribeKV(w, "Stateful:", kw, "yes")
	}

	// Whatever the manifest declared that has no field above, so a key dtctl
	// does not model still reaches the user -- without repeating the fields
	// already printed.
	if len(action.Extra) > 0 {
		rendered, err := yaml.Marshal(action.Extra)
		if err != nil {
			return fmt.Errorf("render action manifest: %w", err)
		}
		fmt.Fprintln(w)
		output.FprintDescribeSection(w, "Additional manifest keys:")
		for _, line := range strings.Split(strings.TrimRight(string(rendered), "\n"), "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

	return nil
}
