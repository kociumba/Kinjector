<!-- # <p align="center">Welcome to</p> -->

<!-- <p align="center">
    <img src="kinjector.svg" alt="Kinjector" title="kinjector logo">
</p> -->

<!-- --- -->

# Kinjector

<!-- ## Overview -->

> [!CAUTION]
> Kinjector may get flagged as malware by Microsoft Defender.

Kinjector is an easy to use modern DLL injector written in go.

This is a Windows only program. Linux support is not planned for now.

<!-- Ktool is a part of my K suite of tools that so far includes:
- [Ktool](https://github.com/kociumba/ktool)
- [Ksorter](https://github.com/kociumba/ksorter)
- [Kinjector](https://github.com/kociumba/Kinjector) - this repo
- [Kpixel](https://github.com/kociumba/kpixel) -->

<!-- This is a project I started working on because I find these things interesting and thought it would be nice if I had an
injector I could trust.

Once this project becomes more mature I will write a wiki for it like I did for Ktool
but for now all the necessary information is here. -->

## Usage

> [!NOTE]
> The soop script is outdated and most likely won't work due to changes in the repo.
<details>
    <summary> Outdated scoop script </summary>
    This script will compile Kinjector on your machine this may take up to 10 minutes the first time around.

    The recommended way to install Kinjector is via the [scoop install file](https://raw.githubusercontent.com/kociumba/Kinjector/main/Kinjector.json).  

    - install the windows package manager [scoop](https://scoop.sh)
    - use `scoop install https://raw.githubusercontent.com/kociumba/Kinjector/main/Kinjector.json` to install Kinjector
    - to update Kinjector, use `scoop uininstall Kinjector` and again `scoop install https://raw.githubusercontent.com/kociumba/Kinjector/main/Kinjector.json`

    Alternatively you can always compile it yourself from source 
</details>

---
Since github actions now automate the build just download the newest version of Kinjector from the [github releases](https://github.com/kociumba/Kinjector/releases)

This is more efficient than the old scoop script and doesn't require any dependencies on your machine.



<!-- ## Notes

This program may be blocked by Microsoft Defender and other antiviruses as it accesses the list of running processes and the kernel32.dll api.

Those things are required for the app to work and the only way to whitelist this app in antiviruses would be to sign it
unfortunately I don't have a certificate, and it's not easy to get one. -->

## Credits

This project uses [fyne](https://github.com/fyne-io/fyne) for it's GUI
and [zenity](https://github.com/ncruces/zenity) for native system dialogs where needed.

Full list of used packages and licenses can be found in the app.
