// DLL Injector
package main

import (
	"flag"
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

var (
	dbg = flag.Bool("dbg", false, "")
)

func trimFilePath(path string) string {

	clog.Info("Selected dll: " + path)

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
	flag.Parse()

	userSelection := &UserSelection{}
	appIcon := fyne.NewStaticResource(resourceIconPng.StaticName, resourceIconPng.StaticContent)
	pathToKinjector, err := os.Executable()
	if err != nil {
		clog.Fatal(err)
	}

	pathToKinjector, err = filepath.EvalSymlinks(pathToKinjector)
	if err != nil {
		clog.Fatal(err)
	}

	if *dbg {
		clog.SetLevel(clog.DebugLevel)
	}

	// set up logger output
	// f, err := os.OpenFile(pathToKinjector+"/log.txt", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	// if err != nil {
	// 	clog.Fatal(err)
	// }
	// defer f.Close()
	// f.WriteString("\n\n")

	// Turn off file logging couse i don't wanna deal with it
	// clog.SetOutput(f)

	clog.Info("Starting dll injector...")

	// force the dark theme
	os.Setenv("FYNE_THEME", "dark")

	// app setup
	a := app.New()
	clog.Debug("app created")

	a.Settings().SetTheme(&injectorTheme{})
	clog.Debug("app theme set")

	w := a.NewWindow("Kinjector")

	w.SetFixedSize(true)

	w.Resize(fyne.NewSize(500, 500))
	clog.Debug("app window created and resized to 500x500")

	w.CenterOnScreen()
	clog.Debug("app window centered on screen")

	w.SetIcon(appIcon)
	clog.Debug("app window icon set")

	// create the system tray menu
	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("Kinjector",

			// register the show window in tray menu
			fyne.NewMenuItem("Show", func() {
				clog.Info("Bringing window back out of system tray")
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

	w.SetContent(widget.NewLabel("Kinjector System Tray"))
	w.SetCloseIntercept(func() {
		clog.Info("Minimizing window into system tray")
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

		// log the user selection
		for _, processName := range userSelection.processNames {
			if processName == s {
				clog.Info("Selected process: " + s)
				break
			}
		}
		// clog.Info(s)
	}

	// create the credits button
	credits := widget.NewButtonWithIcon("Show credits", theme.InfoIcon(), func() {
		CreditsWindow(fyne.CurrentApp(), fyne.NewSize(800, 400)).Show()
	})

	// register text displays
	dllDisplay := widget.NewLabelWithStyle("Dll selected: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// errorDisplay := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	injection := widget.NewActivity()

	// create the app layout
	//
	//
	clog.Info("Creating GUI")
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
						injection.Start()
						if err != nil {
							dialog.NewError(err, w).Show()
							// errorDisplay.SetText(err.Error())
							clog.Warn(err)
							injection.Stop()
						} else {
							dialog.NewInformation("Success", "Injected into "+userSelection.SelectedProc+" !", w).Show()
							// errorDisplay.SetText("Injected into " + userSelection.SelectedProc + " !")
							injection.Stop()
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

	clog.Info("Running...")
	w.ShowAndRun()
}
