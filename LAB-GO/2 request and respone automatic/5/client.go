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



const (
    defaultServerIP = "192.168.2.50:8080"
)

type Config struct {
    ServerIP    string
    SearchPath  string
    FilterExts  bool
    MatchFiles  bool
    MatchFolders bool
}

func parseArgs() Config {
    config := Config{
        ServerIP: defaultServerIP,
    }

    for i := 1; i < len(os.Args); i++ {
        arg := os.Args[i]
        switch {
        case arg == "--help" || arg == "-h":
            printHelp()
            os.Exit(0)
        case arg == "--ext":
            config.FilterExts = true
        case arg == "--file":
            config.MatchFiles = true
        case arg == "--folder":
            config.MatchFolders = true
        case strings.HasPrefix(arg, "--"):
            fmt.Printf("Unknown flag: %s\n", arg)
            os.Exit(1)
        case config.ServerIP == defaultServerIP:
            config.ServerIP = arg + ":8080"
        default:
            config.SearchPath = arg
        }
    }

    if !config.MatchFiles && !config.MatchFolders {
        fmt.Println("Error: Must specify at least one of --file or --folder flags")
        printHelp()
        os.Exit(1)
    }

    return config
}


func printHelp() {
    fmt.Println(`Usage: ./client [SERVER_IP] [SEARCH_PATH] [FLAGS]
    
Arguments:
    SERVER_IP    Optional. IP address of the server (default: 192.168.2.50)
    SEARCH_PATH  Optional. Path to search for files (default: Documents folder)

Flags:
    --file      Enable file pattern matching
    --folder    Enable folder pattern matching
    --ext       Only process .cpp, .py, and .c files (requires --file)
    --help, -h  Show this help message

Examples:
    ./client 192.168.1.2 --file
    ./client 192.168.1.2 --folder
    ./client 192.168.1.2 --file --folder
    ./client 192.168.1.2 --file --ext
    ./client --file --folder "D:\Data\Projects"`)
}

func main() {
    config := parseArgs()

    hostname, err := os.Hostname()
    if err != nil {
        fmt.Printf("Error getting hostname: %v\n", err)
        return
    }

    for {
        fmt.Println("\nWaiting for server connection...")
        conn, err := connectWithRetry(config.ServerIP)
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

            searchPath := config.SearchPath
            if searchPath == "" {
                homeDir, err := os.UserHomeDir()
                if err != nil {
                    fmt.Printf("Error getting home directory: %v\n", err)
                    return
                }
                searchPath = filepath.Join(homeDir, "Documents")
            }

            err = searchAndSendFiles(searchPath, patterns, conn, hostname)
            if err != nil {
                fmt.Printf("Error during file operations: %v\n", err)
            }
        }()

        fmt.Println("Server connection closed. Waiting for next session...")
        time.Sleep(9 * time.Second)
    }
}




func isValidExtension(path string) bool {
    ext := strings.ToLower(filepath.Ext(path))
    return ext == ".cpp" || ext == ".py" || ext == ".c"
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
    config := parseArgs()
    filesFound := false
    matchedFolders := make(map[string]bool)

    // First pass: identify matching folders
    if config.MatchFolders {
        filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
            if err != nil || !info.IsDir() {
                return nil
            }
            if containsPattern(filepath.Base(path), patterns) {
                matchedFolders[path] = true
                filesFound = true
                fmt.Printf("Found matching folder: %s\n", path)
            }
            return nil
        })
    }

    // Second pass: process files and folders
    err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil
        }

        if level := strings.Count(path[len(rootPath):], string(os.PathSeparator)); level > 5 {
            return filepath.SkipDir
        }

        // Check if this path is inside a matched folder
        inMatchedFolder := false
        if config.MatchFolders {
            for folderPath := range matchedFolders {
                if strings.HasPrefix(path, folderPath) {
                    inMatchedFolder = true
                    break
                }
            }
        }

        // Handle folders
        if info.IsDir() {
            if config.MatchFolders && containsPattern(filepath.Base(path), patterns) {
                relPath, err := filepath.Rel(rootPath, path)
                if err != nil {
                    return err
                }

                fileInfo := FileInfo{
                    RelativePath: relPath,
                    Content:      nil,
                    ClientIP:     getLocalIP(),
                    Username:     hostname,
                }

                encoder := json.NewEncoder(conn)
                if err := encoder.Encode(fileInfo); err != nil {
                    return fmt.Errorf("error sending folder info: %v", err)
                }
            }
            return nil
        }

        // Handle files
        shouldSendFile := false
        if config.MatchFiles {
            // Send file if its name matches pattern
            shouldSendFile = containsPattern(filepath.Base(path), patterns)
        }
        if config.MatchFolders {
            // Send file if it's inside a matched folder
            shouldSendFile = shouldSendFile || inMatchedFolder
        }

        if shouldSendFile {
            if config.FilterExts && !isValidExtension(path) {
                return nil
            }

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
        return nil
    })

    if !filesFound {
        fmt.Println("No matching files or folders found")
    }

    return err
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