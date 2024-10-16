package main

import (
	"flag"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
	clog "github.com/charmbracelet/log"
	"github.com/davecgh/go-spew/spew"
	ps "github.com/mitchellh/go-ps"
	"github.com/ncruces/zenity"
)

var (
	dbg = flag.Bool("dbg", false, "")

	configDir           string
	configFile          string
	settingsFile        string
	userSelection       = &UserSelection{}
	suppressSuggestions bool
	procSelect          *xwidget.CompletionEntry
	selectDllButton     *widget.Button
)

func init() {
	d, err := os.UserConfigDir()
	if err != nil {
		clog.Fatal(err)
	}

	configDir = filepath.Join(d, "kinjector")
	configFile = filepath.Join(configDir, "config.json")
	settingsFile = filepath.Join(configDir, "settings.json")

	err = os.MkdirAll(configDir, 0750)
	if err != nil {
		clog.Fatal(err)
	}

	// settingsSelection.loadSettingsFromFile()
}

type InjectionProfile struct {
	Name         string `json:"name"`
	SelectedProc string `json:"selectedProc"`
	SelectedDll  string `json:"selectedDll"`
	DllFile      string `json:"dllFile"`
}

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

func performUnload(userSelection *UserSelection, w fyne.Window) {
	err := Unloader(userSelection)
	if err != nil {
		dialog.NewError(err, w).Show()
		clog.Warn(err)
	} else {
		dialog.NewInformation("Success", "Unloaded from "+userSelection.SelectedProc+" !", w).Show()
	}
}

func main() {
	flag.Parse()

	err := userSelection.loadProfilesFromFile()
	if err != nil {
		clog.Warn("Failed to load profiles:", err)
	}
	appIcon := fyne.NewStaticResource(resourceIconPng.StaticName, resourceIconPng.StaticContent)

	if *dbg {
		clog.SetLevel(clog.DebugLevel)
	}

	clog.SetReportCaller(true)

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
		m.Label = "Kinjector"
		desk.SetSystemTrayMenu(m)
		desk.SetSystemTrayIcon(appIcon)
	}

	w.SetContent(widget.NewLabel("Kinjector System Tray"))
	w.SetCloseIntercept(func() {
		// INFO: Makes sure the app actually closes if the user has the minimize to system tray option disabled
		if !settingsSelection.MinimizeToTray {
			w.Close()
			a.Quit()
			return
		}
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
	procSelect = xwidget.NewCompletionEntry(userSelection.processNames)

	procSelect.OnChanged = func(s string) {
		if !suppressSuggestions {
			userSelection.processNames, err = ProcSnapshot()
			if err != nil {
				clog.Fatal(err)
			}
			matchingProcesses := []string{}
			userSelection.SelectedProc = s // keep this here because of case sensitivity
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
		}
		suppressSuggestions = false
	}

	// create the credits button
	credits := widget.NewButtonWithIcon("Show credits", theme.InfoIcon(), func() {
		CreditsWindow(fyne.CurrentApp(), fyne.NewSize(800, 400)).Show()
	})

	injection := widget.NewActivity()

	// Create the unsafe unload checkbox
	unsafeUnloadCheck := widget.NewCheck("Unsafe Unload (Use with caution)", func(checked bool) {
		userSelection.UnsafeUnload = checked
	})

	// Damn this shit stupid
	selectDllButton = widget.NewButton("", func() {})

	selectDllButton = widget.NewButton("Select DLL", func() {
		userSelection.SelectedDll, err = zenity.SelectFile(zenity.Filename(os.ExpandEnv("$HOME")), zenity.FileFilter{Patterns: []string{"*.dll"}})
		userSelection.DllFile = trimFilePath(userSelection.SelectedDll)
		if userSelection.DllFile != "" {
			selectDllButton.SetText("DLL: " + userSelection.DllFile)
		} else {
			selectDllButton.SetText("Select DLL")
		}
	})

	// Create the main injection form
	injectionForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Select process", Widget: procSelect},
			{Text: "Select DLL", Widget: selectDllButton},
			{Text: "Unsafe Unload", Widget: unsafeUnloadCheck},
		},
		OnSubmit: func() {
			// dialog.NewConfirm(
			// 	"Inject ?",
			// 	fmt.Sprintf("Inject %s with %s ?", userSelection.SelectedProc, userSelection.DllFile),
			// 	func(b bool) {
			// 		if b {
			// 			err := Inject(userSelection)
			// 			injection.Start()
			// 			if err != nil {
			// 				dialog.NewError(err, w).Show()
			// 				clog.Warn(err)
			// 				injection.Stop()
			// 			} else {
			// 				dialog.NewInformation("Success", "Injected into "+userSelection.SelectedProc+" !", w).Show()
			// 				injection.Stop()
			// 			}
			// 		}
			// 	},
			// 	w,
			// ).Show()

			// INFO: no more dialog, to make the shortcut actually usable
			err := Inject(userSelection)
			injection.Start()
			if err != nil {
				dialog.NewError(err, w).Show()
				clog.Warn(err)
				injection.Stop()
			} else {
				dialog.NewInformation("Success", "Injected into "+userSelection.SelectedProc+" !", w).Show()
				injection.Stop()
			}
		},
		OnCancel: func() {
			// INFO: here to disable unloading if the user has the setting off
			if !settingsSelection.AllowUnload {
				dialog.NewInformation(
					"Settings blocked dll unloading",
					"Application settings do not allow unloading right now. Change this in the settings tab.",
					w,
				).Show()
				return
			}
			if userSelection.UnsafeUnload {
				dialog.NewConfirm(
					"Unsafe Unload",
					"Warning: Unsafe unload may cause memory leaks or crash the target process. Proceed?",
					func(confirm bool) {
						if confirm {
							performUnload(userSelection, w)
						}
					},
					w,
				).Show()
			} else {
				performUnload(userSelection, w)
			}
		},
		SubmitText: "Inject",
		// CancelText: func() string {
		// 	if settingsSelection.AllowUnload {
		// 		return "Unload"
		// 	} else {
		// 		return ""
		// 	}(),
		CancelText: "Unload",
	}

	injectShortcutText := canvas.NewText("inject: CTRL+SHIFT+J", color.RGBA{150, 150, 150, 255})
	injectShortcutText.TextSize = 12
	injectShortcutText.TextStyle.Italic = true
	injectShortcutText.Alignment = fyne.TextAlignCenter
	quitShortcutText := canvas.NewText("quit: CTRL+Q", color.RGBA{150, 150, 150, 255})
	quitShortcutText.TextSize = 12
	quitShortcutText.TextStyle.Italic = true
	quitShortcutText.Alignment = fyne.TextAlignCenter

	injectionTab := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Manage Injection", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			injectionForm,
		),

		container.NewVBox(
			injectShortcutText,
			quitShortcutText,
		),
		nil,
		nil,
	)

	initProfiles(w)

	initSettings()

	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Injection", injectionTab),
		container.NewTabItem("Profiles", profileTab),
		container.NewTabItem("Settings", settingsTab),
	)

	// Create a container for the tabs and the bottom buttons
	content := container.NewBorder(nil,
		container.NewVBox(
			widget.NewSeparator(),
			container.NewHBox(
				widget.NewButtonWithIcon("Quit", theme.CancelIcon(), func() {
					w.Close()
					a.Quit()
				}),
				credits,
			),
		),
		nil, nil,
		tabs,
	)

	w.SetContent(content)
	w.Canvas().AddShortcut(
		&desktop.CustomShortcut{
			KeyName:  fyne.KeyJ,
			Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift,
		},
		func(shortcut fyne.Shortcut) {
			injectionForm.OnSubmit()
		})

	w.Canvas().AddShortcut(
		&desktop.CustomShortcut{
			KeyName:  fyne.KeyQ,
			Modifier: fyne.KeyModifierControl,
		},
		func(shortcut fyne.Shortcut) {
			w.Close()
			a.Quit()
		})

	// Useless for the end user but cool for showcasing it in the readme 😎
	w.Canvas().AddShortcut(
		&desktop.CustomShortcut{
			KeyName:  fyne.KeyF2,
			Modifier: fyne.KeyModifierControl,
		},
		func(shortcut fyne.Shortcut) {
			fileName := fmt.Sprintf("%s-%s.png", w.Title(), time.Now().Format("2006-01-02-15-04-05"))
			img := w.Canvas().Capture()
			kinjector, err := os.Executable()
			if err != nil {
				clog.Error("Error saving screenshot", "err", err)
			}
			outFile, err := os.Create(filepath.Join(filepath.Dir(kinjector), fileName))
			if err != nil {
				clog.Error("Error saving screenshot", "err", err)
			} else {
				defer outFile.Close()
				err = png.Encode(outFile, img)
				if err != nil {
					clog.Error("Error saving screenshot", "err", err)
				} else {
					clog.Info("Screenshot saved as", "file", fileName)
				}
			}
		})

	w.Canvas().AddShortcut(
		&desktop.CustomShortcut{
			KeyName:  fyne.KeyD,
			Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift,
		},
		func(shortcut fyne.Shortcut) {
			spew.Dump(userSelection)
		})

	clog.Info("Running...")
	w.ShowAndRun()
}
