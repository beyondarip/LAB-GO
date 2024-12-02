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
    "sync"
)

const (
    port       = ":8080"
    bufferSize = 1024
    baseDir    = "received_files"
)

type ClientHandler struct {
    conn     net.Conn
    hostname string
    clientDir string
}

func NewClientHandler(conn net.Conn) *ClientHandler {
    return &ClientHandler{
        conn: conn,
    }
}

func (ch *ClientHandler) setupClientDirectory() error {
    // Read hostname
    reader := bufio.NewReader(ch.conn)
    hostname, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("error reading hostname: %v", err)
    }
    ch.hostname = hostname[:len(hostname)-1] // Remove newline

    // Create client directory
    ch.clientDir = filepath.Join(baseDir, ch.hostname)
    if err := os.MkdirAll(ch.clientDir, 0755); err != nil {
        return fmt.Errorf("error creating client directory: %v", err)
    }
    
    log.Printf("Connected client: %s", ch.hostname)
    return nil
}

func (ch *ClientHandler) receiveFile() error {
    reader := bufio.NewReader(ch.conn)

    // Read filename
    filename, err := reader.ReadString('\n')
    if err != nil {
        if err == io.EOF {
            return io.EOF
        }
        return fmt.Errorf("error reading filename: %v", err)
    }
    filename = filename[:len(filename)-1] // Remove newline

    // Read filesize
    sizeStr, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("error reading file size: %v", err)
    }
    size, err := strconv.ParseInt(sizeStr[:len(sizeStr)-1], 10, 64)
    if err != nil {
        return fmt.Errorf("error parsing file size: %v", err)
    }

    // Create file
    filePath := filepath.Join(ch.clientDir, filename)
    file, err := os.Create(filePath)
    if err != nil {
        return fmt.Errorf("error creating file: %v", err)
    }
    defer file.Close()

    // Receive file content
    buffer := make([]byte, bufferSize)
    var received int64
    for received < size {
        n, err := reader.Read(buffer)
        if err != nil && err != io.EOF {
            return fmt.Errorf("error receiving file data: %v", err)
        }
        if n == 0 {
            break
        }
        if _, err := file.Write(buffer[:n]); err != nil {
            return fmt.Errorf("error writing to file: %v", err)
        }
        received += int64(n)
    }

    log.Printf("Received file from %s: %s (%d bytes)", ch.hostname, filename, received)
    return nil
}

func (ch *ClientHandler) handle() {
    defer ch.conn.Close()

    if err := ch.setupClientDirectory(); err != nil {
        log.Printf("Error setting up client: %v", err)
        return
    }

    for {
        if err := ch.receiveFile(); err != nil {
            if err == io.EOF {
                log.Printf("Client %s disconnected", ch.hostname)
                return
            }
            log.Printf("Error receiving file from %s: %v", ch.hostname, err)
            return
        }
    }
}

func main() {
    // Set up logging
    log.SetFlags(log.LstdFlags | log.Lshortfile)

    // Create base directory
    if err := os.MkdirAll(baseDir, 0755); err != nil {
        log.Fatalf("Error creating base directory: %v", err)
    }

    // Start TCP server
    listener, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatalf("Error starting server: %v", err)
    }
    defer listener.Close()

    log.Printf("Server listening on port %s", port)

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
            handler := NewClientHandler(conn)
            handler.handle()
        }()
    }
}