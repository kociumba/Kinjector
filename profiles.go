package main

import (
	"encoding/json"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var (
	profileName       = widget.NewEntry()
	profileTab        *fyne.Container
	dllDisplay        *widget.Label
	loadProfileSelect *widget.Select
	updateProfileList func()
	profileForm       *widget.Form
)

func initProfiles(w fyne.Window) {
	profileName.SetPlaceHolder("Profile Name")

	// register text displays
	dllDisplay = widget.NewLabelWithStyle("Dll selected: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// errorDisplay := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	loadProfileSelect = widget.NewSelect([]string{}, func(name string) {
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

	updateProfileList = func() {
		var names []string
		for _, profile := range userSelection.Profiles {
			names = append(names, profile.Name)
		}
		loadProfileSelect.Options = names
	}

	profileForm = &widget.Form{
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

	updateProfileList()

	profileTab = container.NewVBox(
		widget.NewLabelWithStyle("Profile Management", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		profileForm,
	)
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
