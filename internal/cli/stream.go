package cli

import (
	"fmt"

	"github.com/constructspace/loom/internal/core"
	"github.com/spf13/cobra"
)

func newStreamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stream",
		Short: "Manage streams",
	}

	cmd.AddCommand(
		newStreamCreateCmd(),
		newStreamListCmd(),
		newStreamSwitchCmd(),
		newStreamInfoCmd(),
	)

	return cmd
}

func newStreamCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new stream (fork from current)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vault, err := core.OpenVault(projectDir)
			if err != nil {
				return err
			}
			defer vault.Close()

			activeName, _ := vault.Streams.ActiveName()
			stream, err := vault.Streams.Fork(activeName, args[0])
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created stream %q (forked from %q at seq %d)\n",
				stream.Name, activeName, stream.ForkSeq)
			return nil
		},
	}
}

func newStreamListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all streams",
		RunE: func(cmd *cobra.Command, args []string) error {
			vault, err := core.OpenVault(projectDir)
			if err != nil {
				return err
			}
			defer vault.Close()

			streams, err := vault.Streams.List()
			if err != nil {
				return err
			}

			activeName, _ := vault.Streams.ActiveName()

			for _, s := range streams {
				marker := " "
				if s.Name == activeName {
					marker = "*"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %-20s seq %-6d %-8s %s\n",
					marker, s.Name, s.HeadSeq, s.Status, formatTimeAgo(s.UpdatedAt))
			}

			return nil
		},
	}
}

func newStreamSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch active stream",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vault, err := core.OpenVault(projectDir)
			if err != nil {
				return err
			}
			defer vault.Close()

			if err := vault.Streams.SetActive(args[0]); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Switched to stream %q\n", args[0])
			return nil
		},
	}
}

func newStreamInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show stream details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vault, err := core.OpenVault(projectDir)
			if err != nil {
				return err
			}
			defer vault.Close()

			s, err := vault.Streams.GetByName(args[0])
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Stream:    %s\n", s.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "ID:        %s\n", s.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Head:      seq %d\n", s.HeadSeq)
			fmt.Fprintf(cmd.OutOrStdout(), "Status:    %s\n", s.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Created:   %s\n", s.CreatedAt)

			if s.ParentID != "" {
				parent, err := vault.Streams.GetByID(s.ParentID)
				if err == nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Forked from: %s at seq %d\n", parent.Name, s.ForkSeq)
				}
			}

			return nil
		},
	}
}
