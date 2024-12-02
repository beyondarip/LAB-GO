package main

import (
    "fmt"
    "os"
    "path/filepath"
)

func main() {
    // Check if we have enough arguments
    if len(os.Args) < 3 {
        fmt.Println("Usage: program <first_argument> <second_argument>")
        fmt.Println("Example: program UAS 2024")
        os.Exit(1)
    }

    // Get arguments
    arg1 := os.Args[1]
    arg2 := os.Args[2]

    // Determine base path
    basePath := getBasePath()

    // Create folders for classes A to I
    classes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
    for _, class := range classes {
        folderName := fmt.Sprintf("%s_%s_Kelas%s", arg1, arg2, class)
        fullPath := filepath.Join(basePath, folderName)
        
        err := createFolder(fullPath)
        if err != nil {
            fmt.Printf("Error creating folder %s: %v\n", folderName, err)
            continue
        }
        fmt.Printf("Created folder: %s\n", fullPath)
    }
}

func getBasePath() string {
    // If arguments provided, use first argument as base path
    if len(os.Args) > 3 {
        return os.Args[3]
    }
    
    // Get default Documents folder path
    homeDir, err := os.UserHomeDir()
    if err != nil {
        fmt.Printf("Error getting home directory: %v\n", err)
        os.Exit(1)
    }
    return filepath.Join(homeDir, "Documents")
}

func createFolder(path string) error {
    // Create folder with proper permissions
    err := os.MkdirAll(path, 0755)
    if err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }
    return nil
}