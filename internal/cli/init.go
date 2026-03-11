package cli

import (
	"fmt"

	"github.com/constructspace/loom/internal/core"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a new Loom project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := projectDir
			if len(args) > 0 {
				path = args[0]
			}

			vault, err := core.InitVault(path)
			if err != nil {
				return err
			}
			defer vault.Close()

			if name != "" {
				vault.Config.Project.Name = name
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Initialized Loom in %s\n", vault.ProjectPath)

			// Show detected spaces
			if len(vault.Config.Spaces) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Detected spaces:")
				counts, _ := vault.EntityCount()
				for id, sc := range vault.Config.Spaces {
					count := counts[id]
					adapter := sc.Adapter
					if adapter == "git" {
						adapter = "Git repository"
					} else {
						adapter = sc.Path + "/"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %-10s %4d entities (%s)\n", id, count, adapter)
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No spaces detected. Add spaces with: loom space add <id> <path>")
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Stream: main")

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Project name")

	return cmd
}
