package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new account",
	Long:  "Create a new MangaHub account with username, email, and password",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")

		if username == "" {
			fmt.Print("Username: ")
			fmt.Scanln(&username)
		}
		if email == "" {
			fmt.Print("Email: ")
			fmt.Scanln(&email)
		}

		fmt.Print("Password: ")
		password, _ := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()

		fmt.Print("Confirm password: ")
		confirm, _ := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()

		if string(password) != string(confirm) {
			return fmt.Errorf("passwords do not match")
		}

		body := map[string]string{
			"username": username,
			"email":    email,
			"password": string(password),
		}

		jsonBody, _ := json.Marshal(body)
		serverURL := fmt.Sprintf("http://%s:%d/auth/register",
			viper.GetString("server.host"),
			viper.GetInt("server.http_port"))

		resp, err := http.Post(serverURL, "application/json", bytes.NewReader(jsonBody))
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		if result["success"] == true {
			fmt.Println("✓ Account created successfully!")
			fmt.Printf("  Username: %s\n", username)
			fmt.Printf("  Email: %s\n", email)
			fmt.Println("\nNext: mangahub auth login --username " + username)
		} else {
			errorData := result["error"].(map[string]interface{})
			return fmt.Errorf("registration failed: %v", errorData["message"])
		}

		return nil
	},
}

func init() {
	registerCmd.Flags().String("username", "", "Username")
	registerCmd.Flags().String("email", "", "Email address")
	AuthCmd.AddCommand(registerCmd)
}
