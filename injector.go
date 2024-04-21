// DLL Injector
package main

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
	clog "github.com/charmbracelet/log"
	ps "github.com/mitchellh/go-ps"
	"github.com/ncruces/zenity"
)

type UserSelection struct {
	SelectedProc string
	SelectedDll  string
	DllFile      string
	processNames []string
}

func trimFilePath(path string) string {
	_, fileName := filepath.Split(path)
	return fileName
}

// get a snapshot of the current running procesess
//
// convert it into a list of process names
func ProcSnapshot() ([]string, error) {
	procList, err := ps.Processes()
	if err != nil {
		clog.Fatal(err)
	}

	var processNames []string
	for _, process := range procList {
		processNames = append(processNames, process.Executable())
	}
	return processNames, err
}

func main() {
	userSelection := &UserSelection{}
	appIcon := fyne.NewStaticResource(resourceIconPng.StaticName, resourceIconPng.StaticContent)

	// app setup
	a := app.New()
	a.Settings().SetTheme(&injectorTheme{})
	w := a.NewWindow("dll injector")
	w.Resize(fyne.NewSize(500, 500))
	w.CenterOnScreen()
	w.SetIcon(appIcon)

	// create the system tray menu
	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("dll injector",

			// register the show window in tray menu
			fyne.NewMenuItem("Show", func() {
				w.Show()
			}),

			// register the inject current in tray menu
			fyne.NewMenuItem("Inject w/ current settings", func() {
				err := Inject(userSelection)
				if err != nil {
					zenity.Error(err.Error())
				} else {
					zenity.Info("Injected into " + userSelection.SelectedProc + " !")
				}
			}),
		)
		desk.SetSystemTrayMenu(m)
		desk.SetSystemTrayIcon(appIcon)
	}

	w.SetContent(widget.NewLabel("Fyne System Tray"))
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	// get a snapshot of the current running procesess
	// convert it into a list of process names
	initialProcessNames, err := ProcSnapshot()
	if err != nil {
		dialog.NewError(err, w).Show()
		clog.Fatal(err)
	}

	// initial proc list
	// not really nescessary
	userSelection.processNames = initialProcessNames

	// register the process selection input
	procSelect := xwidget.NewCompletionEntry(userSelection.processNames)

	// update the process name suggestions on the fly
	procSelect.OnChanged = func(s string) {
		userSelection.processNames, err = ProcSnapshot()
		if err != nil {
			clog.Fatal(err)
		}
		matchingProcesses := []string{}
		userSelection.SelectedProc = s // keep this here couse of case sensitivity
		s = strings.ToLower(s)
		for _, processName := range userSelection.processNames {
			if strings.Contains(strings.ToLower(processName), s) {
				matchingProcesses = append(matchingProcesses, processName)
			}
		}
		procSelect.SetOptions(matchingProcesses)
		procSelect.ShowCompletion()
		clog.Info(s)
	}

	// create the credits button
	credits := widget.NewButtonWithIcon("Show credits", theme.InfoIcon(), func() {
		CreditsWindow(fyne.CurrentApp(), fyne.NewSize(800, 400)).Show()
	})

	// register text displays
	dllDisplay := widget.NewLabelWithStyle("Dll selected: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// errorDisplay := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// create the app layout
	//
	//
	w.SetContent(container.NewVBox(

		// process slection
		widget.NewLabelWithStyle("Select the process to inject: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		procSelect,
		widget.NewSeparator(),

		// dll selection
		widget.NewButtonWithIcon("Select dll to load", theme.FolderOpenIcon(), func() {
			userSelection.SelectedDll, err = zenity.SelectFile(zenity.Filename(os.ExpandEnv("$HOME")), zenity.FileFilter{Patterns: []string{"*.dll"}})
			userSelection.DllFile = trimFilePath(userSelection.SelectedDll)
			// if err != nil {
			// 	zenity.Error(err.Error())
			// }
			dllDisplay.SetText("Dll selected: " + userSelection.DllFile)
		}),

		// display the sleected dll
		dllDisplay,
		widget.NewSeparator(),

		// triggers the injection
		widget.NewButtonWithIcon("Inject", theme.ConfirmIcon(), func() {
			dialog.NewConfirm(
				"Inject ?",
				"Inject "+userSelection.SelectedProc+" with "+userSelection.DllFile+" ?",
				func(b bool) {
					if b {
						err := Inject(userSelection)
						if err != nil {
							dialog.NewError(err, w).Show()
							// errorDisplay.SetText(err.Error())
							clog.Warn(err)
						} else {
							dialog.NewInformation("Success", "Injected into "+userSelection.SelectedProc+" !", w).Show()
							// errorDisplay.SetText("Injected into " + userSelection.SelectedProc + " !")
						}
					}
				},
				w,
			).Show()
		}),
		// display injection status
		// errorDisplay,
		// quit button
		widget.NewSeparator(),
		widget.NewButtonWithIcon("Quit", theme.CancelIcon(), func() {
			w.Close()
			a.Quit()
		}),
		widget.NewSeparator(),
		credits,
	))

	w.ShowAndRun()
}
