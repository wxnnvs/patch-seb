package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sqweek/dialog"
)

type Release struct {
	Assets []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func main() {
	if !isAdmin() {
		requestAdmin()
	}

	urls := []string{
		"https://wxnnvs.ftp.sh/un-seb/dlls/390/SafeExamBrowser.Browser.dll",
		"https://wxnnvs.ftp.sh/un-seb/dlls/390/SafeExamBrowser.Configuration.dll",
	}

	for _, url := range urls {
		err := downloadFile(url)
		if err != nil {
			dialog.Message("%s", fmt.Sprintf("Failed to download %s: %v\n", url, err)).Title("Download Error").Error()
		}
	}

	movedFiles := []string{}
	for _, url := range urls {
		fileName := filepath.Base(url)
		tempDir := os.TempDir()
		tempFilePath := filepath.Join(tempDir, fileName)

		destinationDir := "C:\\Program Files\\SafeExamBrowser\\Application"
		destinationFilePath := filepath.Join(destinationDir, fileName)

		err := os.MkdirAll(destinationDir, 0755)
		if err != nil {
			dialog.Message("%s", fmt.Sprintf("Failed to create directory %s: %v\n", destinationDir, err)).Title("Directory Error").Error()
			continue
		}

		err = os.Rename(tempFilePath, destinationFilePath)
		if err != nil {
			dialog.Message("%s", fmt.Sprintf("Failed to move %s to %s: %v\n", tempFilePath, destinationFilePath, err)).Title("Move Error").Error()
		} else {
			movedFiles = append(movedFiles, destinationFilePath)
		}
	}

	if len(movedFiles) > 0 {
		dialog.Message("%s", fmt.Sprintf("Successfully removed SEB Hijack from the following files:\n%s", strings.Join(movedFiles, "\n"))).Title("Success").Info()
	} else {
		dialog.Message("%s", "Failed to install.").Title("Error").Info()
	}
}

func isAdmin() bool {
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}

func requestAdmin() {
	dialog.Message("%s", "Please run this program as an administrator.").Title("Admin Required").Error()
	os.Exit(1)
}

func downloadFile(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tempDir := os.TempDir()
	fileName := filepath.Base(url)
	filePath := filepath.Join(tempDir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
