package auth

import "github.com/spf13/cobra"

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  "Register, login, and manage authentication",
}

func init() {
	// Commands added in register.go and login.go
}
