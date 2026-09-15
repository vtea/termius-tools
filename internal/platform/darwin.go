//go:build darwin

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	keychainAccount       = "localKey"
	keychainServiceDirect = "Termius"
	keychainServiceMAS    = "Termius (MAS)"
	masBundleID           = "com.termius.mac"
)

var termiusAppSupportNames = []string{"Termius", "Termius (MAS)"}

type Darwin struct{}

func New() *Darwin { return &Darwin{} }

func (d *Darwin) DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法确定用户主目录：%w", err)
	}
	return pickMacDataDir(home, masTermiusInstalled()), nil
}

func macDataDirCandidates(home string) []string {
	var cands []string
	appSupport := filepath.Join(home, "Library", "Application Support")
	for _, name := range termiusAppSupportNames {
		cands = append(cands, filepath.Join(appSupport, name))
	}

	containerIDs := []string{masBundleID}
	seen := map[string]bool{masBundleID: true}

	entries, err := os.ReadDir(filepath.Join(home, "Library", "Containers"))
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			id := e.Name()
			if seen[id] {
				continue
			}
			if strings.Contains(strings.ToLower(id), "termius") {
				containerIDs = append(containerIDs, id)
				seen[id] = true
			}
		}
	}

	for _, id := range containerIDs {
		base := filepath.Join(home, "Library", "Containers", id, "Data", "Library", "Application Support")
		for _, name := range termiusAppSupportNames {
			cands = append(cands, filepath.Join(base, name))
		}
	}
	return cands
}

func looksLikeTermiusData(dir string) bool {
	markers := []string{
		filepath.Join(dir, "Local Storage", "leveldb"),
		filepath.Join(dir, "IndexedDB"),
	}
	for _, m := range markers {
		st, err := os.Stat(m)
		if err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

func pickMacDataDir(home string, masInstalled bool) string {
	var bestData, firstReadable, firstDenied string
	for _, c := range macDataDirCandidates(home) {
		st, err := os.Stat(c)
		if err != nil {
			if isAccessDenied(err) && firstDenied == "" {
				firstDenied = c
			}
			continue
		}
		if !st.IsDir() {
			continue
		}
		if looksLikeTermiusData(c) {
			if bestData == "" {
				bestData = c
			}
			continue
		}
		if firstReadable == "" {
			firstReadable = c
		}
	}
	if bestData != "" {
		return bestData
	}
	if firstReadable != "" {
		return firstReadable
	}
	if firstDenied != "" {
		return firstDenied
	}
	return defaultMacDataDir(home, masInstalled)
}

func defaultMacDataDir(home string, masInstalled bool) string {
	if masInstalled {
		return filepath.Join(home, "Library", "Containers", masBundleID, "Data", "Library", "Application Support", "Termius")
	}
	return filepath.Join(home, "Library", "Application Support", "Termius")
}

func masTermiusInstalled() bool {
	apps := []string{"/Applications/Termius.app"}
	if home, err := os.UserHomeDir(); err == nil {
		apps = append(apps, filepath.Join(home, "Applications", "Termius.app"))
	}
	for _, app := range apps {
		info := filepath.Join(app, "Contents", "Info")
		cmd := exec.Command("defaults", "read", info, "CFBundleIdentifier")
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) == masBundleID {
			return true
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, "Library", "Containers", masBundleID))
	return err == nil
}

func keychainWriteService(dataDir string, masInstalled bool) string {
	if strings.Contains(filepath.ToSlash(dataDir), "/Containers/") {
		return keychainServiceMAS
	}
	if masInstalled {
		return keychainServiceMAS
	}
	return keychainServiceDirect
}

func (d *Darwin) GetKey() (string, error) {
	var last error
	for _, service := range []string{keychainServiceDirect, keychainServiceMAS} {
		cmd := exec.Command("security", "find-generic-password", "-s", service, "-a", keychainAccount, "-w")
		out, err := cmd.Output()
		if err != nil {
			last = err
			continue
		}
		key := strings.TrimSpace(string(out))
		if key != "" {
			return key, nil
		}
	}
	if last == nil {
		last = fmt.Errorf("empty key")
	}
	return "", fmt.Errorf("无法从钥匙串读取 Termius 密钥：%w\n请确认此设备上已使用过 Termius", last)
}

func (d *Darwin) SetKey(key string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("无法确定用户主目录：%w", err)
	}
	mas := masTermiusInstalled()
	service := keychainWriteService(pickMacDataDir(home, mas), mas)

	exec.Command("security", "delete-generic-password", "-s", service, "-a", keychainAccount).Run()

	cmd := exec.Command("security", "add-generic-password", "-s", service, "-a", keychainAccount, "-w", key)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot write Termius key to Keychain: %w", err)
	}
	return nil
}

func (d *Darwin) IsTermiusRunning() bool {
	// 用 -x 精确匹配进程名 "Termius"，避免误匹配本工具自身
	// （termius-tools.app 进程名为 "Termius Tools"）或其它命令行中含 "Termius" 的进程。
	cmd := exec.Command("pgrep", "-x", "Termius")
	return cmd.Run() == nil
}

func (d *Darwin) CloseHint() string {
	return "macOS：按 Cmd+Q 退出 Termius，或运行 `pkill -x Termius`"
}

func (d *Darwin) OSName() string { return "macOS" }

func isAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	if os.IsPermission(err) {
		return true
	}
	return errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES)
}
