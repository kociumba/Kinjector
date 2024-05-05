package main

import (
	"errors"
	"os"
	"syscall"

	clog "github.com/charmbracelet/log"
	ps "github.com/mitchellh/go-ps"
	"golang.org/x/sys/windows"
)

func Unloader(userSelection *UserSelection) error {
	// var dPath string
	var _pId int
	// var pName string

	pName := userSelection.SelectedProc
	dPath := userSelection.SelectedDll
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

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	pHandle, err := windows.OpenProcess(windows.PROCESS_CREATE_THREAD|windows.PROCESS_VM_OPERATION|windows.PROCESS_VM_WRITE|windows.PROCESS_VM_READ|windows.PROCESS_QUERY_INFORMATION, false, uint32(pId))
	if err != nil {
		return err
	}
	clog.Info("Selected process opened")

	return nil
}
