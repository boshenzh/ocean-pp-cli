// Hand-written commands for managing the covered-route portfolio.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	cmd.AddCommand(newRoutesImportFromScheduleCmd(flags))
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

// scheduleRegistryShape mirrors schedule-pp-cli's registry.json on disk.
// Only the fields we consume here are declared; the registry's other fields
// are ignored.
type scheduleRegistryShape struct {
	Routes []struct {
		Name string `json:"name"`
		POL  string `json:"pol"`
		POD  string `json:"pod"`
	} `json:"routes"`
}

func defaultScheduleRegistryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "schedule-pp-cli", "routes.json"), nil
}

func newRoutesImportFromScheduleCmd(flags *rootFlags) *cobra.Command {
	var registryPath string
	cmd := &cobra.Command{
		Use:   "import-from-schedule",
		Short: "Add covered routes by reading schedule-pp-cli's registry and resolving POD ports to ISO3 countries.",
		Long: `Reads schedule-pp-cli's local registry (default
~/.config/schedule-pp-cli/routes.json), maps each lane's POD port to an ISO3
country code via a built-in lookup table, and adds the resulting countries to
the webprofile covered-route list.

Ports the table doesn't recognize are reported back to the caller; you can
extend coverage by adding the country manually with 'routes add'.`,
		Example: `  # Default: read schedule-pp-cli's standard registry path
  webprofile-pp-cli routes import-from-schedule

  # Custom registry path
  webprofile-pp-cli routes import-from-schedule --registry-path /tmp/routes.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := registryPath
			if path == "" {
				p, err := defaultScheduleRegistryPath()
				if err != nil {
					return err
				}
				path = p
			}
			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("schedule-pp-cli registry not found at %s.\n\nFix one of these:\n  1. Install schedule-pp-cli and run 'schedule-pp-cli routes seed' (loads 7 example lanes)\n  2. Or run 'schedule-pp-cli routes add ...' to register your own lanes\n  3. Or pass --registry-path to point at an existing registry", path)
				}
				return fmt.Errorf("read %s: %w", path, err)
			}
			var sched scheduleRegistryShape
			if err := json.Unmarshal(data, &sched); err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			cfg, cfgPath, err := routes.LoadDefault()
			if err != nil {
				return err
			}
			type entry struct {
				Lane string `json:"lane"`
				POD  string `json:"pod"`
				ISO3 string `json:"iso3"`
			}
			var added, alreadyPresent []entry
			var unknown []map[string]string
			for _, r := range sched.Routes {
				iso3, ok := routes.PortToISO3(r.POD)
				if !ok {
					unknown = append(unknown, map[string]string{"lane": r.Name, "pod": r.POD})
					continue
				}
				wasAdded, addErr := cfg.Add(iso3)
				if addErr != nil {
					return fmt.Errorf("add %s (from %s): %w", iso3, r.POD, addErr)
				}
				e := entry{Lane: r.Name, POD: r.POD, ISO3: iso3}
				if wasAdded {
					added = append(added, e)
				} else {
					alreadyPresent = append(alreadyPresent, e)
				}
			}
			if err := cfg.Save(cfgPath); err != nil {
				return err
			}
			result := map[string]any{
				"registry_path":      path,
				"lanes_in_schedule":  len(sched.Routes),
				"added":              added,
				"already_present":    alreadyPresent,
				"unknown_ports":      unknown,
				"covered_total":      len(cfg.Routes.Covered),
				"webprofile_config":  cfgPath,
			}
			if len(unknown) > 0 && !flags.quiet {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %d port(s) not in built-in lookup. Add manually with 'routes add <ISO3>' if you serve them.\n", len(unknown))
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&registryPath, "registry-path", "",
		"Path to schedule-pp-cli registry.json (default: ~/.config/schedule-pp-cli/routes.json)")
	return cmd
}
