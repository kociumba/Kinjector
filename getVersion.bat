@echo off
setlocal

:: Set default version
set "version=0.0.0"

:: Try to get version from git tags
for /f "usebackq tokens=*" %%i in (`git describe --tags --abbrev=0 2^>nul`) do (
    set "version=%%i"
)

:: Remove 'v' prefix if present
if "%version:~0,1%"=="v" (
    set "version=%version:~1%"
)

:: Check if the version is in the correct format (x.y.z)
for /f "tokens=1-3 delims=." %%a in ("%version%") do (
    if "%%a" neq "" if "%%b" neq "" if "%%c" neq "" (
        set "validVersion=1"
    )
)

:: If the version is not valid, check the VERSION environment variable
if not defined validVersion (
    if defined VERSION (
        for /f "tokens=1-3 delims=." %%a in ("%VERSION%") do (
            if "%%a" neq "" if "%%b" neq "" if "%%c" neq "" (
                set "version=%VERSION%"
                set "validVersion=1"
            )
        )
    )
)

:: Output the version, ensuring it's in the correct format or defaults to 0.0.0
if not defined validVersion (
    echo 0.0.0
) else (
    echo %version%
)

endlocal
