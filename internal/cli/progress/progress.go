package progress

import "github.com/spf13/cobra"

var ProgressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Manage reading progress",
	Long:  "Update and view your manga reading progress across all protocols",
}
