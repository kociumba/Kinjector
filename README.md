# <p align="center">Welcome to</p>

<p align="center">
    <img src="kinjector.svg" alt="Kinjector" title="kinjector logo">
</p>

---

# Overview

> [!CAUTION]
> This project **_does_** get flagged as a virus!
> I can't do much about it and so if it bothers you just don't use it.

Kinjector is a GUI .dll injector written in go.

Ktool is a part of my K suite of tools that so far includes:
- [Ktool](https://github.com/kociumba/ktool)
- [Ksorter](https://github.com/kociumba/ksorter)
- [Kinjector](https://github.com/kociumba/Kinjector) - this repo

This is a project I started working on because I find these things interesting and thought it would be nice if I had an
injector I could trust.

Once this project becomes more mature I will write a wiki for it like I did for Ktool
but for now all the necessary information is here.

## Installation

> [!WARNING]
> This script will compile Kinjector on your machine this may take up to 10 minutes the first time around.

The recommended way to install Kinjector is via the [scoop install file](https://raw.githubusercontent.com/kociumba/Kinjector/main/Kinjector.json).  

- install the windows package manager [scoop](https://scoop.sh)
- use `scoop install https://raw.githubusercontent.com/kociumba/Kinjector/main/Kinjector.json` to install Kinjector
- to update Kinjector, use `scoop uininstall Kinjector` and again `scoop install https://raw.githubusercontent.com/kociumba/Kinjector/main/Kinjector.json`

Alternatively you can always compile it yourself from source 

## Notes

This program may be blocked by Microsoft Defender and other antiviruses as it accesses the list of running processes and the kernel32.dll api.

Those things are required for the app to work and the only way to whitelist this app in antiviruses would be to sign it
unfortunately I don't have a certificate, and it's not easy to get one.

Another thing is that signing the app would require me to distribute the app in a compiled state which I don't plan on doing right now
and the current method that compiles the app on the user system via scoop wouldn't be able to sign it.

### Credits

This project uses [fyne](https://github.com/fyne-io/fyne) for it's GUI
and [zenity](https://github.com/ncruces/zenity) for native system dialogs where needed.

Full list of used packages and licenses can be found in the app.