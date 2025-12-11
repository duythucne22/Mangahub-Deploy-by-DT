package library

import "github.com/spf13/cobra"

var LibraryCmd = &cobra.Command{
	Use:   "library",
	Short: "Manage your manga library",
	Long:  "Add, remove, and view manga in your personal library",
}
