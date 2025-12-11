package manga

import "github.com/spf13/cobra"

var MangaCmd = &cobra.Command{
	Use:   "manga",
	Short: "Manga search and information commands",
	Long:  "Search for manga, view details, and browse the catalog",
}
