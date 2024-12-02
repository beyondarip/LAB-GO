package main

import (
    "bufio"
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

const (
    PORT        = ":8080"
    BUFFER_SIZE = 1024
)

type FileMetadata struct {
    Name string
    Size int64
}

func main() {
    // Setup logging
    logFile, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal(err)
    }
    defer logFile.Close()
    logger := log.New(logFile, "", log.LstdFlags)

    // Create storage directory
    storageDir := "received_files"
    if err := os.MkdirAll(storageDir, 0755); err != nil {
        logger.Fatal("Error creating storage directory:", err)
    }

    // Start TCP server
    listener, err := net.Listen("tcp", PORT)
    if err != nil {
        logger.Fatal("Error starting server:", err)
    }
    defer listener.Close()

    logger.Printf("Server listening on port %s", PORT)
    fmt.Printf("Server started. Listening on port %s\n", PORT)

    // Handle incoming connections
    for {
        conn, err := listener.Accept()
        if err != nil {
            logger.Printf("Error accepting connection: %v", err)
            continue
        }

        // Handle each client in a goroutine
        go handleClient(conn, storageDir, logger)
    }
}

func handleClient(conn net.Conn, storageDir string, logger *log.Logger) {
    defer conn.Close()

    clientAddr := conn.RemoteAddr().String()
    logger.Printf("New connection from %s", clientAddr)

    // Create client-specific directory with timestamp
    timestamp := time.Now().Format("2006-01-02_15-04-05")
    clientDir := filepath.Join(storageDir, fmt.Sprintf("%s_%s", clientAddr, timestamp))
    if err := os.MkdirAll(clientDir, 0755); err != nil {
        logger.Printf("Error creating client directory: %v", err)
        return
    }

    // Read metadata
    reader := bufio.NewReader(conn)
    metadataStr, err := reader.ReadString('\n')
    if err != nil {
        logger.Printf("Error reading metadata from %s: %v", clientAddr, err)
        return
    }

    metadata, err := parseMetadata(metadataStr)
    if err != nil {
        logger.Printf("Error parsing metadata from %s: %v", clientAddr, err)
        return
    }

    // Create and receive file
    filePath := filepath.Join(clientDir, metadata.Name)
    if err := receiveFile(conn, filePath, metadata, logger); err != nil {
        logger.Printf("Error receiving file from %s: %v", clientAddr, err)
        return
    }

    logger.Printf("Successfully received file %s from %s", metadata.Name, clientAddr)
}

func parseMetadata(metadataStr string) (FileMetadata, error) {
    parts := strings.Split(strings.TrimSpace(metadataStr), "|")
    if len(parts) != 2 {
        return FileMetadata{}, fmt.Errorf("invalid metadata format")
    }

    size, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        return FileMetadata{}, fmt.Errorf("invalid file size")
    }

    return FileMetadata{
        Name: parts[0],
        Size: size,
    }, nil
}

func receiveFile(conn net.Conn, filePath string, metadata FileMetadata, logger *log.Logger) error {
    file, err := os.Create(filePath)
    if err != nil {
        return fmt.Errorf("error creating file: %v", err)
    }
    defer file.Close()

    buffer := make([]byte, BUFFER_SIZE)
    var totalReceived int64

    for totalReceived < metadata.Size {
        n, err := conn.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("error reading from connection: %v", err)
        }

        if n > 0 {
            if _, err := file.Write(buffer[:n]); err != nil {
                return fmt.Errorf("error writing to file: %v", err)
            }
            totalReceived += int64(n)
        }
    }

    if totalReceived != metadata.Size {
        return fmt.Errorf("received size (%d) doesn't match expected size (%d)", totalReceived, metadata.Size)
    }

    return nil
}