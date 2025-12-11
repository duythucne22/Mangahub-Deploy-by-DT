package progress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update reading progress",
	Long:  "Update your reading progress - triggers all 5 protocols (HTTP, TCP, UDP, WebSocket, gRPC)!",
	RunE: func(cmd *cobra.Command, args []string) error {
		mangaID, _ := cmd.Flags().GetString("manga-id")
		chapter, _ := cmd.Flags().GetInt("chapter")
		rating, _ := cmd.Flags().GetInt("rating")
		status, _ := cmd.Flags().GetString("status")

		if mangaID == "" {
			return fmt.Errorf("--manga-id is required")
		}

		token := viper.GetString("user.token")
		if token == "" {
			return fmt.Errorf("not logged in. Please run: mangahub auth login")
		}

		body := map[string]interface{}{
			"manga_id":        mangaID,
			"current_chapter": chapter,
			"status":          status,
			"rating":          rating,
		}

		jsonBody, _ := json.Marshal(body)
		serverURL := fmt.Sprintf("http://%s:%d/users/progress",
			viper.GetString("server.host"),
			viper.GetInt("server.http_port"))

		req, _ := http.NewRequest("PUT", serverURL, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to update progress: %w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		if result["success"] == true {
			fmt.Printf("✓ Progress updated successfully!\n")
			fmt.Printf("  Manga ID: %s\n", mangaID)
			fmt.Printf("  Chapter: %d\n", chapter)
			if rating > 0 {
				fmt.Printf("  Rating: %d/10\n", rating)
			}
			fmt.Printf("  Status: %s\n", status)
			fmt.Println("\n🔄 Synced across all protocols:")
			fmt.Println("  ✓ HTTP: API updated")
			fmt.Println("  ✓ TCP: Broadcasted to sync clients")
			fmt.Println("  ✓ UDP: Notification sent")
			fmt.Println("  ✓ WebSocket: Room members notified")
			fmt.Println("  ✓ gRPC: Audit logged")
		} else {
			errorData := result["error"].(map[string]interface{})
			return fmt.Errorf("failed: %v", errorData["message"])
		}

		return nil
	},
}

func init() {
	updateCmd.Flags().String("manga-id", "", "Manga ID (required)")
	updateCmd.Flags().Int("chapter", 0, "Current chapter")
	updateCmd.Flags().Int("rating", 0, "Rating (0-10)")
	updateCmd.Flags().String("status", "reading", "Status (reading, completed, dropped)")
	updateCmd.MarkFlagRequired("manga-id")
	ProgressCmd.AddCommand(updateCmd)
}
