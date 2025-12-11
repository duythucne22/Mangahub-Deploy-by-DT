package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Display current MangaHub CLI configuration and connection settings",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("MangaHub Configuration:")
		fmt.Println("")
		fmt.Printf("Server:\n")
		fmt.Printf("  Host: %s\n", viper.GetString("server.host"))
		fmt.Printf("  HTTP Port: %d\n", viper.GetInt("server.http_port"))
		fmt.Printf("  TCP Port: %d\n", viper.GetInt("server.tcp_port"))
		fmt.Printf("  UDP Port: %d\n", viper.GetInt("server.udp_port"))
		fmt.Printf("  gRPC Port: %d\n", viper.GetInt("server.grpc_port"))
		fmt.Println("")

		username := viper.GetString("user.username")
		token := viper.GetString("user.token")

		if username != "" {
			fmt.Printf("User:\n")
			fmt.Printf("  Username: %s\n", username)
			if token != "" {
				if len(token) > 20 {
					fmt.Printf("  Token: %s...\n", token[:20])
				} else {
					fmt.Printf("  Token: %s\n", token)
				}
				fmt.Printf("  Status: ✓ Logged in\n")
			} else {
				fmt.Printf("  Status: ✗ Not logged in\n")
			}
		} else {
			fmt.Printf("User: Not logged in\n")
			fmt.Printf("  Run 'mangahub auth login' to authenticate\n")
		}
	},
}

func init() {
	ConfigCmd.AddCommand(showCmd)
}
