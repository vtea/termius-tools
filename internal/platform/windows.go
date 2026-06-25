//go:build windows

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	advapi32        = syscall.NewLazyDLL("advapi32.dll")
	procCredReadW   = advapi32.NewProc("CredReadW")
	procCredWriteW  = advapi32.NewProc("CredWriteW")
	procCredDeleteW = advapi32.NewProc("CredDeleteW")
	procCredFree    = advapi32.NewProc("CredFree")
)

const (
	credTypeGeneric  = 1
	credPersistLocal = 2
)

type credential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

type Windows struct{}

func New() *Windows { return &Windows{} }

func (w *Windows) DataDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("无法确定用户主目录：%w", err)
		}
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	return filepath.Join(appData, "Termius"), nil
}

func (w *Windows) GetKey() (string, error) {
	targetName, _ := syscall.UTF16PtrFromString("Termius/localKey")
	var cred *credential

	ret, _, err := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetName)),
		uintptr(credTypeGeneric),
		0,
		uintptr(unsafe.Pointer(&cred)),
	)
	if ret == 0 {
		return "", fmt.Errorf("无法从 Windows 凭据管理器读取 Termius 密钥：%v\n请确认此设备上已使用过 Termius", err)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(cred)))

	blob := unsafe.Slice(cred.CredentialBlob, cred.CredentialBlobSize)
	return string(blob), nil
}

func (w *Windows) SetKey(key string) error {
	targetName, _ := syscall.UTF16PtrFromString("Termius/localKey")
	userName, _ := syscall.UTF16PtrFromString("localKey")

	procCredDeleteW.Call(
		uintptr(unsafe.Pointer(targetName)),
		uintptr(credTypeGeneric),
		0,
	)

	blob := []byte(key)
	cred := credential{
		Type:               credTypeGeneric,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     &blob[0],
		Persist:            credPersistLocal,
		UserName:           userName,
	}

	ret, _, err := procCredWriteW.Call(
		uintptr(unsafe.Pointer(&cred)),
		0,
	)
	if ret == 0 {
		return fmt.Errorf("cannot write Termius key to Windows Credential Manager: %v", err)
	}
	return nil
}

func (w *Windows) IsTermiusRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq Termius.exe", "/NH")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Termius.exe")
}

func (w *Windows) CloseHint() string {
	return "Windows：从系统托盘关闭 Termius，或运行 `taskkill /IM Termius.exe`"
}

func (w *Windows) OSName() string { return "Windows" }
