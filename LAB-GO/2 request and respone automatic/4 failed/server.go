package main

import (
    "bufio"
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "path/filepath"
    "strings"
    "sync"
)

// Server represents the file transfer server
type Server struct {
    port            string
    targetFolders   []string
    baseStoragePath string
    mu             sync.Mutex
}

// NewServer creates a new server instance
func NewServer(targetFolders []string) *Server {
    return &Server{
        port:            ":8080",
        targetFolders:   targetFolders,
        baseStoragePath: "received_files",
    }
}

// Start initializes and runs the server
func (s *Server) Start() error {
    listener, err := net.Listen("tcp", s.port)
    if err != nil {
        return fmt.Errorf("failed to start server: %v", err)
    }
    defer listener.Close()

    log.Printf("Server listening on port %s", s.port)
    log.Printf("Target folders: %v", s.targetFolders)

    // Create base storage directory
    if err := os.MkdirAll(s.baseStoragePath, 0755); err != nil {
        return fmt.Errorf("failed to create storage directory: %v", err)
    }

    var wg sync.WaitGroup
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Error accepting connection: %v", err)
            continue
        }

        wg.Add(1)
        go func() {
            defer wg.Done()
            if err := s.handleConnection(conn); err != nil {
                log.Printf("Error handling connection: %v", err)
            }
        }()
    }
}

// handleConnection processes a single client connection
func (s *Server) handleConnection(conn net.Conn) error {
    defer conn.Close()

    reader := bufio.NewReader(conn)
    
    // Read header
    header, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read header: %v", err)
    }

    // Parse header
    parts := strings.Split(strings.TrimSpace(header), "|")
    if len(parts) != 4 {
        return fmt.Errorf("invalid header format")
    }

    clientIP := parts[0]
    hostname := parts[1]
    relPath := parts[2]
    originalPath := parts[3]

    // Check if the file is in a target folder
    if !s.isTargetFolder(originalPath) {
        return fmt.Errorf("file not in target folder: %s", originalPath)
    }

    // Create client-specific directory
    clientDir := filepath.Join(s.baseStoragePath, fmt.Sprintf("%s_%s", clientIP, hostname))
    if err := os.MkdirAll(clientDir, 0755); err != nil {
        return fmt.Errorf("failed to create client directory: %v", err)
    }

    // Create full path for file
    destPath := filepath.Join(clientDir, relPath)
    if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
        return fmt.Errorf("failed to create directories: %v", err)
    }

    // Create and write file
    s.mu.Lock()
    file, err := os.Create(destPath)
    s.mu.Unlock()
    if err != nil {
        return fmt.Errorf("failed to create file: %v", err)
    }
    defer file.Close()

    // Copy file content
    if _, err := io.Copy(file, reader); err != nil {
        return fmt.Errorf("failed to write file: %v", err)
    }

    log.Printf("Received file from %s (%s): %s", clientIP, hostname, relPath)
    return nil
}

// isTargetFolder checks if the file path contains any of the target folder names
func (s *Server) isTargetFolder(path string) bool {
    lowercasePath := strings.ToLower(path)
    for _, folder := range s.targetFolders {
        if strings.Contains(lowercasePath, strings.ToLower(folder)) {
            return true
        }
    }
    return false
}

func main() {
    // Get target folders from command line arguments
    targetFolders := []string{"Kosong"}
    if len(os.Args) > 1 {
        targetFolders = os.Args[1:]
    }

    // Create and start server
    server := NewServer(targetFolders)
    if err := server.Start(); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}