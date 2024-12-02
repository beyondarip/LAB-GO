package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Get current user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting user home directory: %v\n", err)
		return
	}

	// Directories to clean
	dirsToClean := []string{
		filepath.Join(homeDir, "Downloads"),
		filepath.Join(homeDir, "Documents"),
		filepath.Join(homeDir, "Desktop"),
		filepath.Join(homeDir, "Pictures"),
	}

	// Critical directories and files to exclude
	excludePatterns := []string{
		".vscode",
		"AppData",
		"Links",
		"Videos",
		"Favorites",
		"Contacts",
		"3D Objects",
		"Saved Games",
		"Searches",
		"OneDrive",
		"system.ini",
		"desktop.ini",
		".gitconfig",
		".ssh",
	}

	// File extensions to protect
	protectedExtensions := []string{
		".sys",
		".dll",
		".exe",
		".msi",
		".ini",
	}

	for _, dir := range dirsToClean {
		cleanDirectory(dir, excludePatterns, protectedExtensions)
	}
}

func cleanDirectory(dirPath string, excludePatterns, protectedExtensions []string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", dirPath, err)
		return
	}

	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())

		// Skip if matches exclude patterns
		if isExcluded(entry.Name(), excludePatterns) {
			fmt.Printf("Skipping protected item: %s\n", path)
			continue
		}

		// Skip if file has protected extension
		if hasProtectedExtension(entry.Name(), protectedExtensions) {
			fmt.Printf("Skipping protected file type: %s\n", path)
			continue
		}

		// Remove file or directory
		if err := os.RemoveAll(path); err != nil {
			fmt.Printf("Error removing %s: %v\n", path, err)
		} else {
			fmt.Printf("Successfully removed: %s\n", path)
		}
	}
}

func isExcluded(name string, excludePatterns []string) bool {
	name = strings.ToLower(name)
	for _, pattern := range excludePatterns {
		if strings.Contains(name, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func hasProtectedExtension(name string, protectedExtensions []string) bool {
	nameLower := strings.ToLower(name)
	for _, ext := range protectedExtensions {
		if strings.HasSuffix(nameLower, strings.ToLower(ext)) {
			return true
		}
	}
	return false
}


// buatkan dalam bahasa golang

// buatkan script untuk menghapus semua file di Folder user Downloads user Document, dan beberapa banyak lagi jadi taruh dalam array, dan saya ingin ini dynamic dengan mengetahui nama user pc nya jadi saya tidak perlu "C:/Users/SAQ 41/", hapus semua file dan folder dalam 1 maxdepth itu, dan jangan hapus file penting jadi buat yang exlcude juga, dan juga buat jangan hapus file crusial program / windows seperti .vscode, "Links", "AppData", "Videos", yang seperti itu bawaan dari windows jadi jangan di hapus foldernya. 

// buat code dengan yang best practice dan profesional dalam hal apa pun jangan ada kesalahan



// buatkan dalam bahasa golang

// buatkan script untuk menghapus semua file di Folder user Downloads user Document, dan beberapa banyak lagi jadi taruh dalam array, dan saya ingin ini dynamic dengan mengetahui nama user pc nya jadi saya tidak perlu "C:/Users/SAQ 41/", hapus semua file dan folder dalam 1 maxdepth itu, dan jangan hapus file penting jadi buat yang exlcude juga, dan juga buat jangan hapus file crusial program / windows seperti .vscode, "Links", "AppData", "Videos", yang seperti itu bawaan dari windows jadi jangan di hapus foldernya. 

// semua yang di  atas berada pada partisi C: ,
//  saya mau juga hapus semua folder dan file yang ada di D:

// buat code dengan yang best practice dan profesional dalam hal apa pun jangan ada kesalahan


// tolong lihat pada windows yang pada "C:Users/SAQ 41/ umumnya, apa saja yang penting dan bawaan dari windows yang tidak boleh di hapus/


// ====


// buatkan dalam bahasa golang

// buatkan script untuk mengirim file ke pc server, karena script ini akan di jalankan di setiap pc client, mengirimnya menggunakan ip server yaitu "192.168.2.50", yang dikirim berupa file python / c++ yang dikerjakan mahasiswa di lab, untuk pathnya, program perlu mencari di folder Documents, di situ akan ada banyak folder , tapi yang di kirim folder jika nama folder tersebut mempunyai string "Struktur Data"

// buat code dengan yang best practice dan profesional dalam hal apa pun jangan ada kesalahan