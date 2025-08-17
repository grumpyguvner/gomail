package main

import (
	"net/http"
)

// GetStaticFS returns the filesystem for static files
// In production, this would use embed, but for simplicity we'll use disk files
func GetStaticFS() (http.FileSystem, error) {
	return http.Dir("../../webadmin"), nil
}
