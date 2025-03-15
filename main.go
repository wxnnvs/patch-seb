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
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {

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

	button := widget.NewButton("Select", func() {
		patch(sebVersionWidget, patchVersionWidget, label, w)
	})

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

	w.ShowAndRun()
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
			} else {
				// Handle the cancel action
				fmt.Println("Action canceled")
				patchVersionWidget.Selected = ""
			}
		}, w)
	}

	// Update the label
	label.SetText("SEB Version: " + sebVersionSelected + "\nPatch Version: " + patchVersionSelected)
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
