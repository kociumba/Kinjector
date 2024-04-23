package main

import (
	"errors"
	"os"
	"syscall"

	clog "github.com/charmbracelet/log"
	ps "github.com/mitchellh/go-ps"
	"golang.org/x/sys/windows"
)

// Inject injects a DLL into a running process.
//
// Takes a UserSelection parameter containing the process name and DLL path to inject.
// Returns an error if any operation fails.
func Inject(userSlection *UserSelection) error {
	// var dPath string
	var _pId int
	// var pName string

	pName := userSlection.SelectedProc
	dPath := userSlection.SelectedDll
	// flag.StringVar(&pName, "process", "", "Process name to inject to")
	// flag.StringVar(&dPath, "dll", "", "DLL to inject")
	// flag.Parse()
	pList, err := ps.Processes()
	if err != nil {
		return err
	}
	for pI := range pList {
		process := pList[pI]
		if process.Executable() == pName {
			_pId = process.Pid()
			break
		}
	}
	if _pId == 0 {
		// clog.Warn("Process not found (from inject.go)")
		return errors.New("process not found")
	}
	pId := uintptr(_pId)
	if _, err := os.Stat(dPath); errors.Is(err, os.ErrNotExist) {
		return err
	}
	clog.Info("Selected process id: ", "pId", pId)

	kernel32 := windows.NewLazyDLL("kernel32.dll")
	pHandle, err := windows.OpenProcess(windows.PROCESS_CREATE_THREAD|windows.PROCESS_VM_OPERATION|windows.PROCESS_VM_WRITE|windows.PROCESS_VM_READ|windows.PROCESS_QUERY_INFORMATION, false, uint32(pId))
	if err != nil {
		return err
	}
	clog.Info("Selected process opened")

	VirtualAllocEx := kernel32.NewProc("VirtualAllocEx")
	vAlloc, _, err := VirtualAllocEx.Call(uintptr(pHandle), 0, uintptr(len(dPath)+1), windows.MEM_RESERVE|windows.MEM_COMMIT, windows.PAGE_EXECUTE_READWRITE)
	if err != nil {
		clog.Warn(err)
	}
	clog.Info("Memory allocated")

	bPtrDpath, err := windows.BytePtrFromString(dPath)
	if err != nil {
		return err
	}

	Zero := uintptr(0)
	err = windows.WriteProcessMemory(pHandle, vAlloc, bPtrDpath, uintptr(len(dPath)+1), &Zero)
	if err != nil {
		return err
	}
	clog.Info("Allocated memory written")

	LoadLibAddy, err := syscall.GetProcAddress(syscall.Handle(kernel32.Handle()), "LoadLibraryA")
	if err != nil {
		clog.Warn(err)
	}

	tHandle, _, err := kernel32.NewProc("CreateRemoteThread").Call(uintptr(pHandle), 0, 0, LoadLibAddy, vAlloc, 0, 0)
	if err != nil {
		clog.Warn(err)
	}
	defer syscall.CloseHandle(syscall.Handle(tHandle))
	clog.Info("Thread created")
	clog.Info("DLL injected successfully into " + userSlection.SelectedProc + "!")

	return nil
}
