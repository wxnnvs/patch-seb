package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows"
)

const (
	version = 5
)

type Release struct {
	Assets []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func main() {

	if !isAdmin() {
		runMeElevated()
	}

	// create app and window
	a := app.New()
	w := a.NewWindow("SEB Patcher")
	w.Resize(fyne.NewSize(400, 700))

	// title
	title := widget.NewLabel("SEB Patcher")
	title.Alignment = fyne.TextAlignCenter

	// select seb version
	sebVersionClicked := false
	sebVersionLabel := widget.NewLabel("Select your installed SEB version:")
	sebVersion := []string{"3.9.0", "3.8.0", "3.7.1"}
	sebVersionWidget := widget.NewSelect(sebVersion, nil)

	// select patch version
	patchVersion := []string{}

	patchVersionLabel := widget.NewLabel("\nSelect the patch version you want to install:")
	patchVersionWidget := widget.NewSelect(patchVersion, nil)

	label := widget.NewLabel("Selected: ")

	button := widget.NewButton("Install", func() {
		patch(sebVersionWidget, patchVersionWidget, label, w)
	})

	// check for patcher updates
	if checkLatestRelease(w) > version {
		dialog.ShowCustomConfirm("New Update", "Continue", "Cancel",
			widget.NewLabel("A new update is available.\nDo you want to update?"),
			func(b bool) {
				if b {
					fmt.Println("Updating...")
					updating := dialog.NewCustomWithoutButtons("Updating...", widget.NewLabel("Installing newer version.\nPlease wait."), w)
					updating.Show()
					err := upgrade()
					if err != nil {
						fmt.Println("Error:", err)
						// show error
						updating.Hide()
						dialog.ShowCustom("Error", "Close", widget.NewLabel("Failed to update patcher.\n"+err.Error()), w)
					}
				} else {
					// Handle the cancel action
					fmt.Println("Action canceled")
				}
			}, w)
	}

	// Detect the installed version
	installedVersion := detectVersion()

	// Set the selected SEB version to the installed version
	// and update the patch version selector
	if strings.Contains(installedVersion, "Error") {
		fmt.Println(installedVersion)
		dialog.ShowCustomConfirm("Error", "Install now.", "Ignore", widget.NewLabel("No valid SEB installation found."), func(b bool) {
			if b {
				// open the download page
				exec.Command("rundll32", "url.dll,FileProtocolHandler", "https://github.com/SafeExamBrowser/seb-win-refactoring/releases/latest").Start()
				w.Close()
			} else {
				// Handle the cancel action
				fmt.Println("Action canceled")
			}
		}, w)
	} else {
		sebVersionWidget.Selected = installedVersion
		updatePatchSelector(sebVersionWidget, patchVersionWidget)
	}

	// If the seb version changes, update the patch version selector
	sebVersionWidget.OnChanged = func(selected string) {
		if !sebVersionClicked {
			dialog.ShowCustomConfirm("Warning", "Continue", "Cancel",
				widget.NewLabel("Are you sure you\nwant to change this?\nThe default was selected\nbased on your installation files"),
				func(b bool) {
					if b {
						fmt.Println("Continuing with SEB version:", selected)
						updatePatchSelector(sebVersionWidget, patchVersionWidget)
						sebVersionClicked = true
					} else {
						// Handle the cancel action
						fmt.Println("Action canceled")
						sebVersionWidget.Selected = installedVersion
						sebVersionWidget.Refresh()
						updatePatchSelector(sebVersionWidget, patchVersionWidget)
					}
				}, w)
		}
	}

	// Set the window content
	w.SetContent(container.NewVBox(
		title,
		sebVersionLabel,
		sebVersionWidget,
		patchVersionLabel,
		patchVersionWidget,
		button,
		label))

	ShowAndRunWithTask(func() {
		internetError(w)
	}, w)

}

func fetchLatestPatchVersion() string {

	// Fetch the latest release from the GitHub API
	resp, err := http.Get("https://wxnnvs.ftp.sh/un-seb/latest.json")
	if err != nil {
		fmt.Println("Error fetching release:", err)
		return err.Error()
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return err.Error()
	}

	// Define a struct to hold the release data
	var release struct {
		TagName string `json:"tag_name"`
	}

	// Unmarshal the JSON response into the release struct
	if err := json.Unmarshal(body, &release); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return err.Error()
	}

	// Output the tag name of the latest release
	return release.TagName
}

func setPatchVersion(sebVersion string) []string {

	patchVersion := []string{}

	// Fetch the latest patch version
	latestPatchVersion := fetchLatestPatchVersion()
	if strings.Contains(latestPatchVersion, "Error") {
		fmt.Println(latestPatchVersion)
		return patchVersion
	}

	// Fetch the releases from the GitHub API
	// and add the patch versions to the patchVersion slices
	resp, err := http.Get("https://wxnnvs.ftp.sh/un-seb/releases.json")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			var releases []struct {
				TagName string `json:"tag_name"`
			}
			if err := json.Unmarshal(body, &releases); err != nil {
				fmt.Println("Error:", err)
			} else {
				for _, release := range releases {
					if strings.Contains(release.TagName, sebVersion) {
						// Add '(latest)' to the latest patch versions
						if release.TagName == latestPatchVersion {
							patchVersion = append(patchVersion, release.TagName+" (latest)")
						} else if release.TagName == "v3.8.0_b97253e" {
							patchVersion = append(patchVersion, release.TagName+" (latest)") // 3.8.0
						} else if release.TagName == "v3.7.1_98e8089" {
							patchVersion = append(patchVersion, release.TagName+" (latest)") // 3.7.1
						} else {
							patchVersion = append(patchVersion, release.TagName)
						}
					}
				}
			}
		}
	}

	return patchVersion
}

func patch(sebVersionWidget *widget.Select, patchVersionWidget *widget.Select, label *widget.Label, w fyne.Window) {

	internetError(w)

	// Get the selected option
	sebVersionSelected := sebVersionWidget.Selected
	patchVersionSelected := patchVersionWidget.Selected

	latestPatchVersion := fetchLatestPatchVersion()

	if sebVersionSelected == "3.9.0" {
		// do nothing
	} else if sebVersionSelected == "3.8.0" {
		latestPatchVersion = "v3.8.0_b97253e"
	} else if sebVersionSelected == "3.7.1" {
		latestPatchVersion = "v3.7.1_98e8089"
	}

	if strings.Contains(latestPatchVersion, "Error") {
		fmt.Println(latestPatchVersion)
	}

	// Warning if the selected patch version is not the latest
	if !strings.Contains(patchVersionSelected, latestPatchVersion) {
		dialog.ShowCustomConfirm("Warning", "Continue anyway", "Cancel", widget.NewLabel("You are not using the latest patch version."), func(b bool) {
			if b {
				// Continue with the selected patch version
				fmt.Println("Continuing with patch version:", patchVersionSelected)

				// The actual patching process

				// Update the label
				label.SetText("SEB Version: " + sebVersionSelected + "\nPatch Version: " + patchVersionSelected)

				fetching := dialog.NewCustomWithoutButtons("Fetching assets...", widget.NewLabel("Please wait"), w)
				fetching.Show()

				url := "https://api.github.com/repos/wxnnvs/seb-win-bypass/releases/latest"
				resp, err := http.Get(url)
				if err != nil {
					fmt.Println("Error:", err)
					return
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Println("Error:", err)
					return
				}

				var release Release
				err = json.Unmarshal(body, &release)
				if err != nil {
					fmt.Println("Error:", err)
					return
				}

				urls := []string{}
				for _, asset := range release.Assets {
					if !strings.Contains(asset.BrowserDownloadURL, "exe") {
						urls = append(urls, asset.BrowserDownloadURL)
					}
				}
				fetching.Hide()

				// Download the files and move them to the SEB installation directory
				installing := dialog.NewCustomWithoutButtons("Installing...", widget.NewLabel("Please wait"), w)
				installing.Show()

				for _, url := range urls {
					err := downloadFile(url)
					if err != nil {
						fmt.Println("Error:", err)
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
						fmt.Println("Error:", err)
						continue
					}

					err = os.Rename(tempFilePath, destinationFilePath)
					if err != nil {
						fmt.Println("Error:", err)
					} else {
						movedFiles = append(movedFiles, destinationFilePath)
					}
				}

				if len(movedFiles) > 0 {
					fmt.Println("Successfully installed:")
					for _, file := range movedFiles {
						fmt.Println(file)
					}
					installing.Hide()
					dialog.ShowCustom("Success", "Close", widget.NewLabel("Successfully patched your SEB installation!"), w)
					return
				} else {
					dialog.ShowCustom("Error", "Close", widget.NewLabel("Failed to patch SEB!\nAre you connected to the internet?"), w)
					return
				}

			} else {
				// Handle the cancel action
				fmt.Println("Action canceled")
				patchVersionWidget.Selected = ""
				patchVersionWidget.Refresh()
				return
			}
		}, w)
	} else {
		// The actual patching process

		// Update the label
		label.SetText("SEB Version: " + sebVersionSelected + "\nPatch Version: " + patchVersionSelected)

		fetching := dialog.NewCustomWithoutButtons("Fetching assets...", widget.NewLabel("Please wait"), w)
		fetching.Show()

		url := "https://api.github.com/repos/wxnnvs/seb-win-bypass/releases/latest"
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		var release Release
		err = json.Unmarshal(body, &release)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		urls := []string{}
		for _, asset := range release.Assets {
			if !strings.Contains(asset.BrowserDownloadURL, "exe") {
				urls = append(urls, asset.BrowserDownloadURL)
			}
		}
		fetching.Hide()

		// Download the files and move them to the SEB installation directory
		installing := dialog.NewCustomWithoutButtons("Installing...", widget.NewLabel("Please wait"), w)
		installing.Show()

		for _, url := range urls {
			err := downloadFile(url)
			if err != nil {
				fmt.Println("Error:", err)
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
				fmt.Println("Error:", err)
				continue
			}

			err = os.Rename(tempFilePath, destinationFilePath)
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				movedFiles = append(movedFiles, destinationFilePath)
			}
		}

		if len(movedFiles) > 0 {
			fmt.Println("Successfully installed:")
			for _, file := range movedFiles {
				fmt.Println(file)
			}
			installing.Hide()
			dialog.ShowCustom("Success", "Close", widget.NewLabel("Successfully patched your SEB installation!"), w)
			return
		} else {
			dialog.ShowCustom("Error", "Close", widget.NewLabel("Failed to patch SEB!\nAre you connected to the internet?"), w)
			return
		}
	}
}

func generateMD5() string {
	// Open the file
	file, err := os.Open("c:/Program Files/SafeExamBrowser/Application/SafeExamBrowser.Proctoring.dll")
	if err != nil {
		fmt.Println(err)
		return err.Error()
	}
	defer file.Close()

	// Create a new MD5 hash
	hash := md5.New()

	// Copy the file into the hash
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Println("Error:", err)
		return err.Error()
	}

	// Get the 16-byte hash
	hashInBytes := hash.Sum(nil)

	// Convert the bytes to a string
	hashString := hex.EncodeToString(hashInBytes)

	return hashString
}

func detectVersion() string {
	hash := generateMD5()
	if strings.Contains(hash, "Error") {
		return hash
	}
	if hash == "184550b2479cab509b45291381994ec9" {
		return "3.9.0"
	} else if hash == "fc8abcc53d255b5a9de9a9d09c7ee452" {
		return "3.8.0"
	} else if hash == "6d572137fdf86b0386e4f33491eb8ae4" {
		return "3.7.1"
	} else {
		return "Error: Unknown"
	}
}

func updatePatchSelector(sebVersionWidget *widget.Select, patchVersionWidget *widget.Select) {
	patchVersion := setPatchVersion(sebVersionWidget.Selected)
	patchVersionWidget.Options = patchVersion
	patchVersionWidget.Selected = ""
	patchVersionWidget.Refresh()
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

func isAdmin() bool {
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}

func runMeElevated() {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1 //SW_NORMAL

	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(0)
}

func checkInternet() bool {
	// check internet connection
	_, err := http.Get("https://api.github.com")
	return err == nil
}

func internetError(w fyne.Window) {
	if !checkInternet() {
		noInternet := dialog.NewCustomWithoutButtons("Connection failed.", widget.NewLabel("Please connect to the internet."), w)
		noInternet.Show()
		for !checkInternet() {
			time.Sleep(1 * time.Second) // Add a delay to avoid busy waiting
		}
		noInternet.Hide()
		detectVersion()
	}
}

func ShowAndRunWithTask(task func(), w fyne.Window) {
	go task() // Run the task concurrently
	w.ShowAndRun()
}

// check latest release
func checkLatestRelease(w fyne.Window) int {
	internetError(w)

	url := "https://api.github.com/repos/wxnnvs/patch-seb/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	// Parse the release response to get the tag name
	var release struct {
		TagName string `json:"tag_name"`
	}

	// Decode the response JSON
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&release); err != nil {
		return 0
	}

	versionStr := strings.TrimPrefix(release.TagName, "v")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		fmt.Println("Error converting version:", err)
		return 0
	}
	return version
}

func upgrade() error {
	// Step 1: Fetch the latest release info from GitHub
	latestRelease, err := fetchLatestRelease()
	if err != nil {
		return err
	}

	// Step 2: Download the new .exe from the release
	err = downloadExe(latestRelease)
	if err != nil {
		return err
	}

	// Step 3: Schedule the replacement after the current process exits
	err = scheduleReplacement()
	if err != nil {
		return err
	}

	fmt.Println("Upgrade scheduled. Restarting application...")
	return nil
}

func fetchLatestRelease() (string, error) {
	url := "https://api.github.com/repos/wxnnvs/patch-seb/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch release info: %v", err)
	}
	defer resp.Body.Close()

	// Parse the release response to get the asset URL
	var release struct {
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}

	// Decode the response JSON
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode release info: %v", err)
	}

	// Find the .exe asset in the release
	for _, asset := range release.Assets {
		if asset.Name == "patch-seb.exe" {
			return asset.URL, nil
		}
	}

	return "", fmt.Errorf("no .exe asset found in latest release")
}

func downloadExe(downloadURL string) error {
	// Step 2: Download the file
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	// Step 3: Save the .exe to the specified path
	filePath := filepath.Join("./", "patch-seb-temp.exe")
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy content: %v", err)
	}

	fmt.Printf("Downloaded new executable to %s\n", filePath)
	return nil
}

func scheduleReplacement() error {
	// Step 4: Schedule the replacement of the current executable
	// You cannot overwrite the running executable, so you will need to schedule the replacement on exit
	// This can be achieved by running a new process to replace the current .exe with the new one after exit

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not get current executable: %v", err)
	}

	// Create a temporary batch file to handle the copying, deletion of .exe, and deleting the batch file
	batchFilePath := filepath.Join("./", "replace_and_cleanup.bat")
	batchContent := fmt.Sprintf(`
@echo off
timeout /t 3
copy /Y "%s" "%s"
del /f "%s"
start "" "%s"
del /f "%s"
exit
`, filepath.Join("./", "patch-seb-temp.exe"), currentExe, filepath.Join("./", "patch-seb-temp.exe"), "%~dp0patch-seb.exe", batchFilePath)

	err = os.WriteFile(batchFilePath, []byte(batchContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to create batch file: %v", err)
	}

	// Step 5: Run the batch file that will perform the replacement and cleanup
	cmd := exec.Command("cmd", "/C", batchFilePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start batch file command: %v", err)
	}

	// Exit the current process so the new one can replace it
	os.Exit(0)

	return nil
}
