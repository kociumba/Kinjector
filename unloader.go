package main

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	clog "github.com/charmbracelet/log"
	ps "github.com/mitchellh/go-ps"
	"golang.org/x/sys/windows"
)

func TryCommonStopFunctions(pHandle windows.Handle, dllBaseAddr uintptr) bool {
	// Common function names that may stop threads
	stopFunctions := []string{"StopThreads", "StopAll", "Cleanup", "Shutdown"}
	kernel32 := windows.NewLazyDLL("kernel32.dll")
	for _, fnName := range stopFunctions {
		// Attempt to find the function in the DLL
		stopProc := kernel32.NewProc(fnName)
		if stopProc.Find() == nil {
			// Function exists, let's try calling it
			clog.Info("Found stop function in DLL: ", "function", fnName)
			tHandle, _, err := kernel32.NewProc("CreateRemoteThread").Call(
				uintptr(pHandle),
				0,
				0,
				stopProc.Addr(),
				0, // We don't pass any arguments to these functions
				0,
				0,
			)
			if tHandle != 0 && err == nil {
				defer windows.CloseHandle(windows.Handle(tHandle))
				clog.Info("Successfully called stop function: ", "function", fnName)
				return true
			} else {
				clog.Warn("Failed to call stop function", "function", fnName, "err", err)
			}
		}
	}
	// If none of the stop functions worked
	return false
}

func ForcefullyTerminateDllThreads(pHandle windows.Handle, dllBaseAddr, dllEndAddr uintptr) error {
	kernel32 := windows.NewLazyDLL("kernel32.dll")
	terminateThread := kernel32.NewProc("TerminateThread")

	// Create snapshot of all threads in the target process
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)

	var te windows.ThreadEntry32
	te.Size = uint32(unsafe.Sizeof(te))

	// Iterate through the threads
	if err := windows.Thread32First(snapshot, &te); err != nil {
		return err
	}

	for {
		// If the thread belongs to the target process
		if te.OwnerProcessID == uint32(pHandle) {
			threadHandle, err := windows.OpenThread(windows.THREAD_QUERY_INFORMATION|windows.THREAD_TERMINATE, false, te.ThreadID)
			if err != nil {
				clog.Warn("Failed to open thread", "threadID", te.ThreadID)
				continue
			}
			defer windows.CloseHandle(threadHandle)

			// Get thread start address and check if it belongs to the DLL's memory range
			var startAddress uintptr
			ntdll := windows.NewLazyDLL("ntdll.dll")
			ntQueryInformationThread := ntdll.NewProc("NtQueryInformationThread")
			threadInfoClass := 9 // ThreadQuerySetWin32StartAddress
			ntQueryInformationThread.Call(
				uintptr(threadHandle),
				uintptr(threadInfoClass),
				uintptr(unsafe.Pointer(&startAddress)),
				unsafe.Sizeof(startAddress),
				0,
			)

			// Check if the thread's start address falls within the DLL's memory range
			if startAddress >= dllBaseAddr && startAddress <= dllEndAddr {
				clog.Info("Terminating thread started by DLL", "threadID", te.ThreadID)

				// Use TerminateThread via syscall
				ret, _, err := terminateThread.Call(uintptr(threadHandle), 0)
				if ret == 0 {
					clog.Warn("Failed to terminate thread", "threadID", te.ThreadID, "err", err)
					continue
				}
				clog.Info("Thread terminated successfully", "threadID", te.ThreadID)
			}
		}

		// Move to the next thread
		err = windows.Thread32Next(snapshot, &te)
		if err != nil {
			break
		}
	}

	return nil
}

func Unloader(userSelection *UserSelection) error {
	var _pId int
	pName := userSelection.SelectedProc
	dPath := userSelection.SelectedDll

	// Check if the DLL exists
	if _, err := os.Stat(dPath); errors.Is(err, os.ErrNotExist) {
		return err
	}

	// List processes and find the target process by name
	pList, err := ps.Processes()
	if err != nil {
		return err
	}
	for _, process := range pList {
		if process.Executable() == pName {
			_pId = process.Pid()
			break
		}
	}
	if _pId == 0 {
		return errors.New("process not found")
	}
	pId := uintptr(_pId)

	clog.Info("Selected process id: ", "pId", pId)

	// Open the target process with more permissions
	pHandle, err := windows.OpenProcess(windows.EVENT_ALL_ACCESS, false, uint32(pId))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(pHandle)
	clog.Info("Selected process opened")

	// Use CreateToolhelp32Snapshot to get a snapshot of the modules in the process
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE|windows.TH32CS_SNAPMODULE32, uint32(pId))
	if err != nil {
		return errors.New("failed to create snapshot of modules")
	}
	defer windows.CloseHandle(snapshot)

	var me windows.ModuleEntry32
	me.Size = uint32(unsafe.Sizeof(me))

	// Iterate through the modules using Module32First and Module32Next
	if err := windows.Module32First(snapshot, &me); err != nil {
		return errors.New("failed to retrieve first module")
	}

	dName := filepath.Base(dPath)

	// Loop through the modules and check for the DLL
	for {
		moduleName := windows.UTF16ToString(me.Module[:])
		if moduleName == dName {
			clog.Info("Found DLL module handle: ", "dllName", moduleName)

			if userSelection.UnsafeUnload {
				// Try to call common stop functions
				if !TryCommonStopFunctions(pHandle, uintptr(me.ModBaseAddr)) {
					clog.Warn("No stop function found or called successfully, proceeding to forcefully stop threads")
				}

				// Forcefully terminate any threads started by the DLL
				dllEndAddr := uintptr(me.ModBaseAddr) + uintptr(me.ModBaseSize)
				err = ForcefullyTerminateDllThreads(pHandle, uintptr(me.ModBaseAddr), dllEndAddr)
				if err != nil {
					clog.Error("Failed to forcefully terminate threads", "err", err)
					return err
				}
			}

			// Get the address of FreeLibrary
			kernel32 := windows.NewLazyDLL("kernel32.dll")
			FreeLibrary := kernel32.NewProc("FreeLibrary")
			if FreeLibrary.Find() != nil {
				return errors.New("failed to find FreeLibrary function")
			}

			// Create a remote thread to call FreeLibrary
			tHandle, _, err := kernel32.NewProc("CreateRemoteThread").Call(
				uintptr(pHandle),
				0,
				0,
				FreeLibrary.Addr(),
				uintptr(me.ModBaseAddr), // DLL handle to be freed
				0,
				0,
			)
			if tHandle == 0 {
				return errors.New("failed to create remote thread")
			}
			if err != nil && err != syscall.Errno(0) {
				return err
			}
			defer syscall.CloseHandle(syscall.Handle(tHandle))

			clog.Info("DLL unloaded successfully from " + userSelection.SelectedProc + "!")
			return nil
		}

		// Move to the next module in the snapshot
		err = windows.Module32Next(snapshot, &me)
		if err != nil {
			break
		}
	}

	clog.Error("Failed to find DLL in target process", "dPath", dPath, "dName", dName, "pName", pName, "pId", pId, "err", err)

	return errors.New("DLL not found in target process")
}
