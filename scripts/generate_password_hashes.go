package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("Generating bcrypt password hashes with DefaultCost (10):")
	fmt.Println()

	// Admin password
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	fmt.Printf("Password: admin123\nHash: %s\n\n", string(hash))

	// Moderator password
	hash, _ = bcrypt.GenerateFromPassword([]byte("moderator123"), bcrypt.DefaultCost)
	fmt.Printf("Password: moderator123\nHash: %s\n\n", string(hash))

	// Test user password
	hash, _ = bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	fmt.Printf("Password: testpass123\nHash: %s\n\n", string(hash))
}
