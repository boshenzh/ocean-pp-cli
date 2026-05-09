// Hand-written commands for managing the covered-route portfolio.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/internal/routes"

	"github.com/spf13/cobra"
)

func newRoutesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Manage the covered-route portfolio (which countries you can ship to).",
		Long: `Routes are the ISO3 country codes you can serve. fit-score uses this list to
award up to 30 bonus points when an importer is on a covered lane.

Stored at ~/.config/webprofile-pp-cli/routes.toml as plain TOML you can
hand-edit; each subcommand validates and normalizes entries.`,
	}
	cmd.AddCommand(newRoutesShowCmd(flags))
	cmd.AddCommand(newRoutesAddCmd(flags))
	cmd.AddCommand(newRoutesRemoveCmd(flags))
	cmd.AddCommand(newRoutesInitCmd(flags))
	cmd.AddCommand(newRoutesResetCmd(flags))
	cmd.AddCommand(newRoutesPathCmd(flags))
	cmd.AddCommand(newRoutesEditCmd(flags))
	return cmd
}

func newRoutesShowCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "show",
		Short:       "Print the current covered-route list.",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := routes.LoadDefault()
			if err != nil {
				return err
			}
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"config_path": path,
					"covered":     cfg.Routes.Covered,
					"count":       len(cfg.Routes.Covered),
				}, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", path)
			if cfg.IsEmpty() {
				fmt.Fprintln(cmd.OutOrStdout(), "Covered: (none)")
				fmt.Fprintln(cmd.OutOrStdout(), "Hint: run 'webprofile-pp-cli routes init <ISO3...>' to set up.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Covered (%d):\n", len(cfg.Routes.Covered))
			for _, code := range cfg.Routes.Covered {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", code)
			}
			return nil
		},
	}
}

func newRoutesAddCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "add <ISO3...>",
		Short: "Add one or more ISO3 country codes to the covered list.",
		Args:  cobra.MinimumNArgs(1),
		Example: `  webprofile-pp-cli routes add EGY
  webprofile-pp-cli routes add EGY SAU ARE`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := routes.LoadDefault()
			if err != nil {
				return err
			}
			var added, skipped []string
			for _, in := range args {
				ok, err := cfg.Add(in)
				if err != nil {
					return err
				}
				code := strings.ToUpper(strings.TrimSpace(in))
				if ok {
					added = append(added, code)
				} else {
					skipped = append(skipped, code)
				}
			}
			if err := cfg.Save(path); err != nil {
				return err
			}
			if !flags.quiet {
				if len(added) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Added: %s\n", strings.Join(added, ", "))
				}
				if len(skipped) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Already present: %s\n", strings.Join(skipped, ", "))
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Total covered: %d\n", len(cfg.Routes.Covered))
			}
			return nil
		},
	}
}

func newRoutesRemoveCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "remove <ISO3...>",
		Aliases: []string{"rm"},
		Short:   "Remove one or more ISO3 country codes from the covered list.",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := routes.LoadDefault()
			if err != nil {
				return err
			}
			var removed, missing []string
			for _, in := range args {
				code := strings.ToUpper(strings.TrimSpace(in))
				if cfg.Remove(in) {
					removed = append(removed, code)
				} else {
					missing = append(missing, code)
				}
			}
			if err := cfg.Save(path); err != nil {
				return err
			}
			if !flags.quiet {
				if len(removed) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Removed: %s\n", strings.Join(removed, ", "))
				}
				if len(missing) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Not in list: %s\n", strings.Join(missing, ", "))
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Total covered: %d\n", len(cfg.Routes.Covered))
			}
			return nil
		},
	}
}

func newRoutesInitCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init <ISO3...>",
		Short: "Replace the covered list with the given ISO3 codes.",
		Long: `Init overwrites the existing covered list. Use 'add' to append. Useful for
first-time setup or scripted redeployment.

Re-run with --yes if the list is non-empty (safety prompt).`,
		Args:    cobra.MinimumNArgs(1),
		Example: `  webprofile-pp-cli routes init EGY SAU ARE IND PAK DJI YEM --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := routes.LoadDefault()
			if err != nil {
				return err
			}
			existing := len(cfg.Routes.Covered)
			if existing > 0 && !flags.yes {
				return fmt.Errorf("init refused: %d existing routes would be replaced; re-run with --yes to confirm", existing)
			}
			if err := cfg.Init(args); err != nil {
				return err
			}
			if err := cfg.Save(path); err != nil {
				return err
			}
			if !flags.quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "Saved %d routes to %s\n", len(cfg.Routes.Covered), path)
			}
			return nil
		},
	}
}

func newRoutesResetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Clear the covered-route list (re-run with --yes to confirm).",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := routes.LoadDefault()
			if err != nil {
				return err
			}
			if !cfg.IsEmpty() && !flags.yes {
				return fmt.Errorf("reset refused: %d covered routes would be cleared; re-run with --yes to confirm", len(cfg.Routes.Covered))
			}
			cfg.Reset()
			if err := cfg.Save(path); err != nil {
				return err
			}
			if !flags.quiet {
				fmt.Fprintln(cmd.OutOrStdout(), "Covered list cleared.")
			}
			return nil
		},
	}
}

func newRoutesPathCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "path",
		Short:       "Print the path to routes.toml.",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := routes.DefaultPath()
			if err != nil {
				return err
			}
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"path": path}, flags)
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

func newRoutesEditCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open routes.toml in $EDITOR. Validates on save.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := routes.DefaultPath()
			if err != nil {
				return err
			}
			// Make sure the file exists so the editor has something to open.
			if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
				cfg, _, lerr := routes.LoadDefault()
				if lerr != nil {
					return lerr
				}
				if serr := cfg.Save(path); serr != nil {
					return serr
				}
			}
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			ed := exec.Command(editor, path)
			ed.Stdin = os.Stdin
			ed.Stdout = os.Stdout
			ed.Stderr = os.Stderr
			if err := ed.Run(); err != nil {
				return fmt.Errorf("%s exited with error: %w", editor, err)
			}
			if _, err := routes.Load(path); err != nil {
				return fmt.Errorf("config invalid after edit: %w", err)
			}
			if !flags.quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", path)
			}
			return nil
		},
	}
}
