package main

import (
    "encoding/json"
    "fmt"
	"time"
    "io"
    "net"
    "os"
    "path/filepath"
    "sync"
)

type FileInfo struct {
    RelativePath string
    Content      []byte
    ClientIP     string
    Username     string
}

const (
    PORT = ":8080"
    BASE_DIR = "received_files"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: ./server <pattern1> <pattern2> ...")
        return
    }

    patterns := os.Args[1:]
    
    // Create base directory
    if err := os.MkdirAll(BASE_DIR, 0755); err != nil {
        fmt.Printf("Error creating base directory: %v\n", err)
        return
    }

    // Start TCP server
    listener, err := net.Listen("tcp", PORT)
    if err != nil {
        fmt.Printf("Error starting server: %v\n", err)
        return
    }
    defer listener.Close()

    fmt.Printf("Server listening on %s\nAccepted patterns: %v\n", PORT, patterns)

    var wg sync.WaitGroup
    for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Printf("Error accepting connection: %v\n", err)
            continue
        }

        wg.Add(1)
        go handleClient(conn, patterns, &wg)
    }
}

func handleClient(conn net.Conn, patterns []string, wg *sync.WaitGroup) {
    defer conn.Close()
    defer wg.Done()

    clientAddr := conn.RemoteAddr().String()
    fmt.Printf("New connection from: %s\n", clientAddr)

    // Send patterns to client
    encoder := json.NewEncoder(conn)
    if err := encoder.Encode(patterns); err != nil {
        fmt.Printf("Error sending patterns to client %s: %v\n", clientAddr, err)
        return
    }

    // Receive files
    decoder := json.NewDecoder(conn)
    for {
        var fileInfo FileInfo
        err := decoder.Decode(&fileInfo)
        if err == io.EOF {
            break
        }
        if err != nil {
            fmt.Printf("Error receiving file from %s: %v\n", clientAddr, err)
            return
        }

        if err := saveFile(fileInfo); err != nil {
            fmt.Printf("Error saving file from %s: %v\n", clientAddr, err)
            continue
        }

        fmt.Printf("Received file from %s: %s\n", clientAddr, fileInfo.RelativePath)
    }
}

// func saveFile(fileInfo FileInfo) error {
//     // Create client-specific directory
//     clientDir := filepath.Join(BASE_DIR, fileInfo.Username+"_"+fileInfo.ClientIP)
    
//     // Create full path for file
//     fullPath := filepath.Join(clientDir, fileInfo.RelativePath)
    
//     // Ensure directory exists
//     dirPath := filepath.Dir(fullPath)
//     if err := os.MkdirAll(dirPath, 0755); err != nil {
//         return fmt.Errorf("error creating directory structure: %v", err)
//     }

//     // Write file
//     if err := os.WriteFile(fullPath, fileInfo.Content, 0644); err != nil {
//         return fmt.Errorf("error writing file: %v", err)
//     }

//     return nil
// }



func saveFile(fileInfo FileInfo) error {
    // Get current timestamp
    timestamp := time.Now().Format("2006_01_02___15_04_05")
    
    // Create client-specific directory with timestamp
    clientDir := filepath.Join(BASE_DIR, 
        fmt.Sprintf("%s_%s_%s", fileInfo.Username, fileInfo.ClientIP, timestamp))
    
    // Create full path for file
    fullPath := filepath.Join(clientDir, fileInfo.RelativePath)
    
    // Ensure directory exists
    dirPath := filepath.Dir(fullPath)
    if err := os.MkdirAll(dirPath, 0755); err != nil {
        return fmt.Errorf("error creating directory structure: %v", err)
    }

    // Write file
    if err := os.WriteFile(fullPath, fileInfo.Content, 0644); err != nil {
        return fmt.Errorf("error writing file: %v", err)
    }

    return nil
}
