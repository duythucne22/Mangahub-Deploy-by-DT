package library

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your manga library",
	Long:  "View all manga in your library with reading progress",
	RunE: func(cmd *cobra.Command, args []string) error {
		token := viper.GetString("user.token")
		if token == "" {
			return fmt.Errorf("not logged in. Please run: mangahub auth login")
		}

		serverURL := fmt.Sprintf("http://%s:%d/users/library",
			viper.GetString("server.host"),
			viper.GetInt("server.http_port"))

		req, _ := http.NewRequest("GET", serverURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to get library: %w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		if result["success"] == true {
			library := result["data"].([]interface{})

			fmt.Printf("\nYour Library (%d manga):\n\n", len(library))

			for i, item := range library {
				entry := item.(map[string]interface{})
				manga := entry["manga"].(map[string]interface{})
				progress := entry["reading_progress"].(map[string]interface{})

				fmt.Printf("%d. %s\n", i+1, manga["title"].(string))
				fmt.Printf("   Author: %s\n", manga["author"].(string))
				fmt.Printf("   Status: %s\n", progress["status"].(string))
				if chapter, ok := progress["current_chapter"].(float64); ok {
					fmt.Printf("   Progress: Chapter %.0f\n", chapter)
				}
				if rating, ok := progress["rating"].(float64); ok && rating > 0 {
					fmt.Printf("   Rating: %.0f/10\n", rating)
				}
				fmt.Println()
			}
		} else {
			errorData := result["error"].(map[string]interface{})
			return fmt.Errorf("failed: %v", errorData["message"])
		}

		return nil
	},
}

func init() {
	LibraryCmd.AddCommand(listCmd)
}
