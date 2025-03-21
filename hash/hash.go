package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

func main() {
	filePath := "c:/Program Files/SafeExamBrowser/Application/SafeExamBrowser.Proctoring.dll"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Println("Error hashing file:", err)
		return
	}

	hashInBytes := hash.Sum(nil)[:16]
	hashString := fmt.Sprintf("%x", hashInBytes)
	fmt.Println("MD5 hash of file:", hashString)
}
