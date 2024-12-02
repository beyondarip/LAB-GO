package main

import (
    "fmt"
    "net"
    "os"
	"time"
    "path/filepath"
    "strings"
    "encoding/json"
)

type FileInfo struct {
    RelativePath string
    Content      []byte
    ClientIP     string
    Username     string
}
func main() {
    // Handle server IP
    serverIP := "192.168.2.50:8080"
    if len(os.Args) > 1 {
        serverIP = os.Args[1] + ":8080"
    }

    // Handle search path
    var searchPath string
    if len(os.Args) > 2 {
        searchPath = os.Args[2]
    } else {
        // Use default Documents folder
        homeDir, err := os.UserHomeDir()
        if err != nil {
            fmt.Printf("Error getting home directory: %v\n", err)
            return
        }
        searchPath = filepath.Join(homeDir, "Documents")
    }

    // Verify path exists
    if _, err := os.Stat(searchPath); os.IsNotExist(err) {
        fmt.Printf("Search path does not exist: %s\n", searchPath)
        return
    }

    hostname, err := os.Hostname()
    if err != nil {
        fmt.Printf("Error getting hostname: %v\n", err)
        return
    }

    for {
        fmt.Println("\nWaiting for server connection...")
        conn, err := connectWithRetry(serverIP)
        if err != nil {
            fmt.Printf("Failed to connect: %v\n", err)
            time.Sleep(5 * time.Second)
            continue
        }

        func() {
            defer conn.Close()
            
            patterns, err := getPathPatternsFromServer(conn)
            if err != nil {
                fmt.Printf("Error getting patterns from server: %v\n", err)
                return
            }

            fmt.Printf("Processing patterns: %v\n", patterns)

            // docPath, err := getDocumentsPath()
            // if err != nil {
            //     fmt.Printf("Error getting documents path: %v\n", err)
            //     return
            // }
            docPath := searchPath

            err = searchAndSendFiles(docPath, patterns, conn, hostname)
            if err != nil {
                fmt.Printf("Error during file operations: %v\n", err)
            }
        }()

        fmt.Println("Server connection closed. Waiting for next session...")
        time.Sleep(2 * time.Second) // Small delay before next attempt
    }
}

func containsPattern(path string, patterns []string) bool {
    pathLower := strings.ToLower(path)
    for _, pattern := range patterns {
        if strings.Contains(pathLower, strings.ToLower(pattern)) {
            return true
        }
    }
    return false
}

func searchAndSendFiles(rootPath string, patterns []string, conn net.Conn, hostname string) error {
    filesFound := false
    err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil // Skip files we can't access
        }

        // Check depth
        if level := strings.Count(path[len(rootPath):], string(os.PathSeparator)); level > 5 {
            return filepath.SkipDir
        }

        // Check if file has relevant extension and contains pattern
        if !info.IsDir() && 
           (strings.HasSuffix(strings.ToLower(path), ".py") ||
            strings.HasSuffix(strings.ToLower(path), ".cpp") ||
            strings.HasSuffix(strings.ToLower(path), ".c")) {
            
            if containsPattern(path, patterns) {
                filesFound = true
                relPath, err := filepath.Rel(rootPath, path)
                if err != nil {
                    return err
                }

                content, err := os.ReadFile(path)
                if err != nil {
                    fmt.Printf("Error reading file %s: %v\n", path, err)
                    return nil
                }

                fileInfo := FileInfo{
                    RelativePath: relPath,
                    Content:      content,
                    ClientIP:     getLocalIP(),
                    Username:     hostname,
                }

                encoder := json.NewEncoder(conn)
                if err := encoder.Encode(fileInfo); err != nil {
                    return fmt.Errorf("error sending file info: %v", err)
                }

                fmt.Printf("Sent file: %s\n", relPath)
            }
        }
        return nil
    })

    if !filesFound {
        fmt.Println("No matching files found")
    }

    return err
}

// Rest of the helper functions remain the same

func processDirectory(dirPath string, conn net.Conn, rootPath string, hostname string) error {
    return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if !info.IsDir() && (strings.HasSuffix(info.Name(), ".py") || 
                            strings.HasSuffix(info.Name(), ".cpp") ||
                            strings.HasSuffix(info.Name(), ".c")) {
            relPath, err := filepath.Rel(rootPath, path)
            if err != nil {
                return err
            }

            content, err := os.ReadFile(path)
            if err != nil {
                return err
            }

            fileInfo := FileInfo{
                RelativePath: relPath,
                Content:      content,
                ClientIP:     getLocalIP(),
                Username:     hostname,
            }

            encoder := json.NewEncoder(conn)
            if err := encoder.Encode(fileInfo); err != nil {
                return fmt.Errorf("error sending file info: %v", err)
            }

            fmt.Printf("Sent file: %s\n", relPath)
        }
        return nil
    })
}

func connectWithRetry(serverIP string) (net.Conn, error) {
    maxRetries := 100 // Limit retries to prevent infinite loop
    retryCount := 0
    
    for retryCount < maxRetries {
        conn, err := net.Dial("tcp", serverIP)
        if err == nil {
            fmt.Println("Connected to server successfully")
            return conn, nil
        }
        
        retryCount++
        fmt.Printf("Connection attempt %d failed, retrying in 5 seconds...\n", retryCount)
        time.Sleep(5 * time.Second)
    }
    
    return nil, fmt.Errorf("failed to connect after %d attempts", maxRetries)
}

func getPathPatternsFromServer(conn net.Conn) ([]string, error) {
    decoder := json.NewDecoder(conn)
    var patterns []string
    
    if err := decoder.Decode(&patterns); err != nil {
        return nil, fmt.Errorf("error receiving patterns from server: %v", err)
    }
    
    fmt.Printf("Received patterns from server: %v\n", patterns)
    return patterns, nil
}

func getDocumentsPath() (string, error) {
    // For Windows
    home := os.Getenv("USERPROFILE")
    if home == "" {
        // Fallback for other OS
        var err error
        home, err = os.UserHomeDir()
        if err != nil {
            return "", fmt.Errorf("could not find home directory: %v", err)
        }
    }
    
    documentsPath := filepath.Join(home, "Documents")
    if _, err := os.Stat(documentsPath); os.IsNotExist(err) {
        return "", fmt.Errorf("documents folder not found: %v", err)
    }
    
    return documentsPath, nil
}

func getLocalIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return ""
    }
    for _, addr := range addrs {
        if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    return ""
}