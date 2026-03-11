package cli

import (
	"fmt"

	"github.com/constructspace/loom/internal/core"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show project status",
		RunE: func(cmd *cobra.Command, args []string) error {
			vault, err := core.OpenVault(projectDir)
			if err != nil {
				return err
			}
			defer vault.Close()

			stream, err := vault.ActiveStream()
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Project: %s\n", vault.Config.Project.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Stream:  %s (seq %d)\n", stream.Name, stream.HeadSeq)

			// Last checkpoint
			cps, _ := vault.Checkpoints.List(stream.ID, 1)
			if len(cps) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Last checkpoint: %q (%s)\n", cps[0].Title, formatTimeAgo(cps[0].Timestamp))
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No checkpoints yet")
			}

			// Space status
			counts, _ := vault.EntityCount()
			lastCPSeq := vault.Checkpoints.LatestSeq(stream.ID)
			changedCounts, _ := vault.OpReader.CountBySpace(stream.ID, lastCPSeq)

			if len(counts) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "\nSpaces:")
				for spaceID, count := range counts {
					oc := changedCounts[spaceID]
					if oc != nil {
						total := oc.Created + oc.Modified + oc.Deleted
						fmt.Fprintf(cmd.OutOrStdout(), "  %-10s %4d entities   %d changed since checkpoint\n", spaceID, count, total)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "  %-10s %4d entities   0 changed\n", spaceID, count)
					}
				}
			}

			// Pending ops
			totalPending := 0
			for _, oc := range changedCounts {
				totalPending += oc.Created + oc.Modified + oc.Deleted
			}
			if totalPending > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%d operations since last checkpoint\n", totalPending)
			}

			return nil
		},
	}
}
