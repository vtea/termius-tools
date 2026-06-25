//go:build linux

package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Linux struct{}

func New() *Linux { return &Linux{} }

func (l *Linux) DataDir() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("无法确定用户主目录：%w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "Termius"), nil
}

func (l *Linux) GetKey() (string, error) {
	cmd := exec.Command("secret-tool", "lookup", "service", "Termius", "account", "localKey")
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		return strings.TrimSpace(string(out)), nil
	}

	cmd = exec.Command("keyctl", "request", "user", "termius:localKey")
	out, err = cmd.Output()
	if err == nil && len(out) > 0 {
		keyID := strings.TrimSpace(string(out))
		cmd = exec.Command("keyctl", "pipe", keyID)
		out, err = cmd.Output()
		if err == nil && len(out) > 0 {
			return string(out), nil
		}
	}

	dataDir, _ := l.DataDir()
	localState := filepath.Join(dataDir, "Local State")
	data, err := os.ReadFile(localState)
	if err == nil {
		var state map[string]interface{}
		if json.Unmarshal(data, &state) == nil {
			if osc, ok := state["os_crypt"].(map[string]interface{}); ok {
				if key, ok := osc["encrypted_key"].(string); ok && key != "" {
					return key, nil
				}
			}
		}
	}

	return "", fmt.Errorf("无法读取 Termius 密钥：已尝试 secret-tool、keyctl 和 Local State\n请确认此设备上已使用过 Termius")
}

func (l *Linux) SetKey(key string) error {
	cmd := exec.Command("secret-tool", "store", "--label=Termius localKey", "service", "Termius", "account", "localKey")
	cmd.Stdin = strings.NewReader(key)
	if err := cmd.Run(); err == nil {
		return nil
	}

	cmd = exec.Command("keyctl", "add", "user", "termius:localKey", key, "@u")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot store Termius key: tried secret-tool and keyctl: %w\nInstall libsecret-tools or keyutils", err)
	}
	return nil
}

func (l *Linux) IsTermiusRunning() bool {
	// 用 -x 精确匹配进程名 "Termius"，避免误匹配本工具自身
	// 或其它命令行中含 "Termius" 的进程。
	cmd := exec.Command("pgrep", "-x", "Termius")
	return cmd.Run() == nil
}

func (l *Linux) CloseHint() string {
	return "Linux：关闭 Termius，或运行 `pkill -f Termius`"
}

func (l *Linux) OSName() string { return "Linux" }
