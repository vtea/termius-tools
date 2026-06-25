//go:build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Darwin struct{}

func New() *Darwin { return &Darwin{} }

func (d *Darwin) DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法确定用户主目录：%w", err)
	}
	return filepath.Join(home, "Library", "Application Support", "Termius"), nil
}

func (d *Darwin) GetKey() (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", "Termius", "-a", "localKey", "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("无法从钥匙串读取 Termius 密钥：%w\n请确认此设备上已使用过 Termius", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d *Darwin) SetKey(key string) error {
	exec.Command("security", "delete-generic-password", "-s", "Termius", "-a", "localKey").Run()

	cmd := exec.Command("security", "add-generic-password", "-s", "Termius", "-a", "localKey", "-w", key)
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
	return "macOS：按 Cmd+Q 退出，或运行 `pkill -f Termius`"
}

func (d *Darwin) OSName() string { return "macOS" }
