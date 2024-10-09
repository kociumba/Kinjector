// DLL Injector
package main

import (
	"encoding/json"
	"flag"
	"fmt"
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

var (
	dbg = flag.Bool("dbg", false, "")

	configDir    string
	configFile   string
	settingsFile string
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

type UserSelection struct {
	SelectedProc string
	SelectedDll  string
	DllFile      string
	processNames []string
	Profiles     []InjectionProfile
	UnsafeUnload bool
}

func (u *UserSelection) SaveProfile(name string) error {
	profile := InjectionProfile{
		Name:         name,
		SelectedProc: u.SelectedProc,
		SelectedDll:  u.SelectedDll,
		DllFile:      u.DllFile,
	}

	u.Profiles = append(u.Profiles, profile)
	return u.saveProfilesToFile()
}

func (u *UserSelection) LoadProfile(name string) error {
	for _, profile := range u.Profiles {
		if profile.Name == name {
			u.SelectedProc = profile.SelectedProc
			u.SelectedDll = profile.SelectedDll
			u.DllFile = profile.DllFile
			return nil
		}
	}
	return fmt.Errorf("profile not found")
}

func (u *UserSelection) saveProfilesToFile() error {
	data, err := json.Marshal(u.Profiles)
	if err != nil {
		return err
	}

	err = os.WriteFile(configFile, data, 0600)
	if err != nil {
		return err
	}

	// err = u.loadProfilesFromFile()
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (u *UserSelection) loadProfilesFromFile() error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, which is fine for first run
			return nil
		}
		return err
	}

	err = json.Unmarshal(data, &u.Profiles)
	if err != nil {
		return err
	}

	return nil
}

func (u *UserSelection) DeleteProfile(name string) error {
	for i, profile := range u.Profiles {
		if profile.Name == name {
			// Remove the profile from the slice
			u.Profiles = append(u.Profiles[:i], u.Profiles[i+1:]...)
			return u.saveProfilesToFile()
		}
	}
	return fmt.Errorf("profile not found")
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

	userSelection := &UserSelection{}
	err := userSelection.loadProfilesFromFile()
	if err != nil {
		clog.Warn("Failed to load profiles:", err)
	}
	appIcon := fyne.NewStaticResource(resourceIconPng.StaticName, resourceIconPng.StaticContent)
	// pathToKinjector, err := os.Executable()
	// if err != nil {
	// 	clog.Fatal(err)
	// }

	// pathToKinjector, err = filepath.EvalSymlinks(pathToKinjector)
	// if err != nil {
	// 	clog.Fatal(err)
	// }

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
	var suppressSuggestions bool

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

	// register text displays
	dllDisplay := widget.NewLabelWithStyle("Dll selected: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// errorDisplay := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	injection := widget.NewActivity()

	profileName := widget.NewEntry()
	profileName.SetPlaceHolder("Profile Name")

	// Create the unsafe unload checkbox
	unsafeUnloadCheck := widget.NewCheck("Unsafe Unload (Use with caution)", func(checked bool) {
		userSelection.UnsafeUnload = checked
	})

	// Create the unload button with confirmation dialog
	// unloadButton := widget.NewButtonWithIcon("Unload", theme.CancelIcon(), func() {
	// 	if userSelection.UnsafeUnload {
	// 		dialog.NewConfirm(
	// 			"Unsafe Unload",
	// 			"Warning: Unsafe unload may cause memory leaks or crash the target process. Proceed?",
	// 			func(confirm bool) {
	// 				if confirm {
	// 					performUnload(userSelection, w)
	// 				}
	// 			},
	// 			w,
	// 		).Show()
	// 	} else {
	// 		performUnload(userSelection, w)
	// 	}
	// })

	// // create the app layout
	// //
	// //
	// clog.Info("Creating GUI")
	// // Create the main injection tab
	// injectionTab := container.NewVBox(
	// 	widget.NewLabelWithStyle("Select the process to inject: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	// 	procSelect,
	// 	widget.NewSeparator(),
	// 	widget.NewButtonWithIcon("Select dll to load", theme.FolderOpenIcon(), func() {
	// 		userSelection.SelectedDll, err = zenity.SelectFile(zenity.Filename(os.ExpandEnv("$HOME")), zenity.FileFilter{Patterns: []string{"*.dll"}})
	// 		userSelection.DllFile = trimFilePath(userSelection.SelectedDll)
	// 		dllDisplay.SetText("Dll selected: " + userSelection.DllFile)
	// 	}),
	// 	dllDisplay,
	// 	// widget.NewSeparator(),
	// 	widget.NewButtonWithIcon("Inject", theme.ConfirmIcon(), func() {
	// 		dialog.NewConfirm(
	// 			"Inject ?",
	// 			"Inject "+userSelection.SelectedProc+" with "+userSelection.DllFile+" ?",
	// 			func(b bool) {
	// 				if b {
	// 					err := Inject(userSelection)
	// 					injection.Start()
	// 					if err != nil {
	// 						dialog.NewError(err, w).Show()
	// 						clog.Warn(err)
	// 						injection.Stop()
	// 					} else {
	// 						dialog.NewInformation("Success", "Injected into "+userSelection.SelectedProc+" !", w).Show()
	// 						injection.Stop()
	// 					}
	// 				}
	// 			},
	// 			w,
	// 		).Show()
	// 	}),
	// 	// triggers the unloader
	// 	// widget.NewSeparator(),
	// 	widget.NewSeparator(),
	// 	unsafeUnloadCheck,
	// 	unloadButton,
	// )

	// Damn this shit stupid
	selectDllButton := widget.NewButton("", func() {})

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
			dialog.NewConfirm(
				"Inject ?",
				fmt.Sprintf("Inject %s with %s ?", userSelection.SelectedProc, userSelection.DllFile),
				func(b bool) {
					if b {
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
					}
				},
				w,
			).Show()
		},
		OnCancel: func() {
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
		CancelText: "Unload",
	}

	injectionTab := container.NewVBox(
		widget.NewLabelWithStyle("Manage Injection", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		injectionForm,
	)

	// Create the profile management tab
	profileName = widget.NewEntry()
	profileName.SetPlaceHolder("Profile Name")

	loadProfileSelect := widget.NewSelect([]string{}, func(name string) {
		err := userSelection.LoadProfile(name)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			suppressSuggestions = true
			procSelect.SetText(userSelection.SelectedProc)
			dllDisplay.SetText("Dll selected: " + userSelection.DllFile)
			selectDllButton.SetText("DLL: " + userSelection.DllFile)
			dialog.ShowInformation("Loaded profile", "Profile "+userSelection.SelectedProc+" loaded successfully", w)
		}
	})

	updateProfileList := func() {
		var names []string
		for _, profile := range userSelection.Profiles {
			names = append(names, profile.Name)
		}
		loadProfileSelect.Options = names
	}

	profileForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Profile Name", Widget: profileName},
			{Text: "Load Profile", Widget: loadProfileSelect},
		},
		OnSubmit: func() {
			if profileName.Text != "" {
				err := userSelection.SaveProfile(profileName.Text)
				if err != nil {
					dialog.ShowError(err, w)
				} else {
					dialog.ShowInformation("Success", "Profile saved", w)
					updateProfileList()
					profileName.SetText("") // Clear the profile name entry after saving
				}
			} else {
				dialog.ShowInformation("Error", "Please enter a profile name", w)
			}
		},
		OnCancel: func() {
			if loadProfileSelect.Selected == "" {
				dialog.ShowInformation("Error", "Please select a profile to delete", w)
				return
			}
			dialog.NewConfirm(
				"Delete Profile",
				"Are you sure you want to delete the profile '"+loadProfileSelect.Selected+"'?",
				func(confirm bool) {
					if confirm {
						err := userSelection.DeleteProfile(loadProfileSelect.Selected)
						if err != nil {
							dialog.ShowError(err, w)
						} else {
							dialog.ShowInformation("Success", "Profile deleted", w)
							updateProfileList()
							loadProfileSelect.SetSelected("")
						}
					}
				},
				w,
			).Show()
		},
		SubmitText: "Save Profile",
		CancelText: "Delete Profile",
	}

	// saveProfileButton := widget.NewButtonWithIcon("Save Profile", theme.DocumentSaveIcon(), func() {
	// 	if profileName.Text != "" {
	// 		err := userSelection.SaveProfile(profileName.Text)
	// 		if err != nil {
	// 			dialog.ShowError(err, w)
	// 		} else {
	// 			dialog.ShowInformation("Success", "Profile saved", w)
	// 			updateProfileList()
	// 		}
	// 	} else {
	// 		dialog.ShowInformation("Error", "Please enter a profile name", w)
	// 	}
	// })

	// deleteProfileButton := widget.NewButtonWithIcon("Delete Profile", theme.DeleteIcon(), func() {
	// 	if loadProfileSelect.Selected == "" {
	// 		dialog.ShowInformation("Error", "Please select a profile to delete", w)
	// 		return
	// 	}
	// 	dialog.NewConfirm(
	// 		"Delete Profile",
	// 		"Are you sure you want to delete the profile '"+loadProfileSelect.Selected+"'?",
	// 		func(confirm bool) {
	// 			if confirm {
	// 				err := userSelection.DeleteProfile(loadProfileSelect.Selected)
	// 				if err != nil {
	// 					dialog.ShowError(err, w)
	// 				} else {
	// 					dialog.ShowInformation("Success", "Profile deleted", w)
	// 					updateProfileList()
	// 					loadProfileSelect.SetSelected("")
	// 				}
	// 			}
	// 		},
	// 		w,
	// 	).Show()
	// })

	updateProfileList()

	profileTab := container.NewVBox(
		widget.NewLabelWithStyle("Profile Management", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		profileForm,
	)

	// profileTab := container.NewVBox(
	// 	widget.NewLabelWithStyle("Profile Management", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	// 	profileName,
	// 	saveProfileButton,
	// 	widget.NewSeparator(),
	// 	widget.NewLabel("Load Profile:"),
	// 	loadProfileSelect,
	// 	deleteProfileButton,
	// )

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

	clog.Info("Running...")
	w.ShowAndRun()
}
