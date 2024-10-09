package main

import (
	"encoding/json"
	"os"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	clog "github.com/charmbracelet/log"
)

type SettingsSelection struct {
	MinimizeToTray bool
	AllowUnload    bool
}

var (
	settingsSelection = SettingsSelection{}

	unloadCheck  *widget.Check
	sysTrayCheck *widget.Check

	settings    *fyne.Container
	settingsTab *fyne.Container
)

func initSettings() {
	unloadCheck = widget.NewCheck("Allow Unloading (may cause memory leaks and crashes)", func(checked bool) {
		settingsSelection.AllowUnload = checked
		settingsSelection.saveSettingsToFile()
	})

	sysTrayCheck = widget.NewCheck("Minimize Kinjector to system tray on app closing?", func(checked bool) {
		settingsSelection.MinimizeToTray = checked
		settingsSelection.saveSettingsToFile()
	})

	settings = container.NewVBox(
		sysTrayCheck,
		unloadCheck,
		widget.NewButtonWithIcon("Open config directory", theme.FolderIcon(), func() {
			exec.Command("explorer", configDir).Start()
		}),
	)

	settingsTab = container.NewVBox(
		widget.NewLabelWithStyle("Kinjector settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		settings,
	)

	// Load settings from file
	err := settingsSelection.loadSettingsFromFile()
	if err != nil {
		// Handle error (e.g., log it or show a message to the user)
		clog.Error("Error loading settings", "err", err)
	}

	// Update checkbox states
	updateCheckboxStates()
}

func (s *SettingsSelection) saveSettingsToFile() error {
	data, err := json.Marshal(settingsSelection)
	if err != nil {
		return err
	}

	err = os.WriteFile(settingsFile, data, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (s *SettingsSelection) loadSettingsFromFile() error {
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, which is fine for first run
			return nil
		}
		return err
	}

	err = json.Unmarshal(data, &settingsSelection)
	if err != nil {
		return err
	}

	return nil
}

func updateCheckboxStates() {
	unloadCheck.SetChecked(settingsSelection.AllowUnload)
	sysTrayCheck.SetChecked(settingsSelection.MinimizeToTray)
}
