# Makefile

# Define the directory variable. Default to the current directory if not provided.
dir ?= $(CURDIR)

# Alias 'build' to perform all tasks
build: cd $(dir) && go_get && fyne_install && fyne_bundle && fyne_package

# Retrieve dependencies
go_get:
    go get -C $(dir) -d ./...

# Install Fyne command line tool
fyne_install:
    go install fyne.io/fyne/v2/cmd/fyne@latest

# Bundle assets
fyne_bundle:
    fyne bundle -o $(dir)/bundled.go $(dir)/icon.png

# Package the application
fyne_package:
    fyne package -os windows -icon $(dir)/icon.png

.PHONY: build go_get fyne_install fyne_bundle fyne_package