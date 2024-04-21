# Makefile

# Define the directory variable. Default to the current directory if not provided.
dir ?= $(CURDIR)

# Alias 'build' to perform all tasks
build: go_get fyne_install fyne_bundle fyne_package

# Retrieve dependencies
go_get:
	@cd $(dir) && go get -d ./...

# Install Fyne command line tool
fyne_install:
	@cd $(dir) && go install fyne.io/fyne/v2/cmd/fyne@latest

# Bundle assets
fyne_bundle:
	@cd $(dir) && fyne bundle -o bundled.go icon.png

# Package the application
fyne_package:
	@cd $(dir) && fyne package -os windows -icon icon.png -src .

.PHONY: build go_get fyne_install fyne_bundle fyne_package
