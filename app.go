package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	goruntime "runtime"
	"strings"
	"syscall"

	"termius-tools/internal/backup"
	"termius-tools/internal/platform"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
	p   backup.Platform
}

func NewApp() *App {
	return &App{p: platform.New()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type BackupInfoView struct {
	Version   string             `json:"version"`
	CreatedAt string             `json:"createdAt"`
	Hostname  string             `json:"hostname"`
	Files     []backup.FileEntry `json:"files"`
	FileCount int                `json:"fileCount"`
}

type StatusInfo struct {
	OS              string `json:"os"`
	DataDir         string `json:"dataDir"`
	TermiusRunning  bool   `json:"termiusRunning"`
	DataDirExists   bool   `json:"dataDirExists"`
	DataDirReadable bool   `json:"dataDirReadable"`
	DataDirHint     string `json:"dataDirHint"`
	CloseHint       string `json:"closeHint"`
	Version         string `json:"version"`
}

func (a *App) GetStatus() StatusInfo {
	dataDir, _ := a.p.DataDir()
	exists, readable := classifyDataDir(dataDir)
	hint := ""
	if exists && !readable {
		if goruntime.GOOS == "darwin" {
			hint = "请在「系统设置 → 隐私与安全性 → 完全磁盘访问权限」中允许 Termius Tools"
		} else {
			hint = "无法读取 Termius 数据目录，请检查文件权限"
		}
	}
	return StatusInfo{
		OS:              a.p.OSName(),
		DataDir:         dataDir,
		TermiusRunning:  a.p.IsTermiusRunning(),
		DataDirExists:   exists,
		DataDirReadable: readable,
		DataDirHint:     hint,
		CloseHint:       a.p.CloseHint(),
		Version:         AppVersion,
	}
}

func classifyDataDir(path string) (exists, readable bool) {
	_, err := os.Stat(path)
	if err == nil {
		if _, rerr := os.ReadDir(path); isAccessDenied(rerr) {
			return true, false
		}
		return true, true
	}
	if isAccessDenied(err) {
		return true, false
	}
	return false, false
}

func isAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	if os.IsPermission(err) {
		return true
	}
	return errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES)
}

func (a *App) CreateBackup(outPath string, forceIfRunning bool) (*backup.BackupResult, error) {
	if outPath == "" {
		path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "保存 Termius 备份",
			DefaultFilename: defaultBackupName(),
			Filters: []runtime.FileFilter{
				{DisplayName: "Termius 备份 (*.tbk)", Pattern: "*.tbk"},
			},
		})
		if err != nil {
			return nil, err
		}
		if path == "" {
			return nil, fmt.Errorf("cancelled")
		}
		outPath = path
	}
	return backup.Backup(a.p, outPath, forceIfRunning)
}

func (a *App) RestoreBackup(inPath string) (*backup.RestoreResult, error) {
	if inPath == "" {
		path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
			Title: "选择 Termius 备份",
			Filters: []runtime.FileFilter{
				{DisplayName: "Termius 备份 (*.tbk)", Pattern: "*.tbk"},
			},
		})
		if err != nil {
			return nil, err
		}
		if path == "" {
			return nil, fmt.Errorf("cancelled")
		}
		inPath = path
	}
	return backup.Restore(a.p, inPath)
}

func (a *App) InspectBackup(inPath string) (*BackupInfoView, error) {
	if inPath == "" {
		path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
			Title: "选择 Termius 备份",
			Filters: []runtime.FileFilter{
				{DisplayName: "Termius 备份 (*.tbk)", Pattern: "*.tbk"},
			},
		})
		if err != nil {
			return nil, err
		}
		if path == "" {
			return nil, fmt.Errorf("cancelled")
		}
		inPath = path
	}
	info, err := backup.Inspect(inPath)
	if err != nil {
		return nil, err
	}
	return &BackupInfoView{
		Version:   info.Metadata.Version,
		CreatedAt: info.Metadata.CreatedAt.Format("2006-01-02 15:04:05"),
		Hostname:  info.Metadata.Hostname,
		Files:     info.Files,
		FileCount: len(info.Files),
	}, nil
}

func (a *App) RevealInFinder(path string) error {
	if path == "" {
		return fmt.Errorf("no path specified")
	}
	switch goruntime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	case "windows":
		return exec.Command("explorer", "/select,", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func defaultBackupName() string {
	hostname, _ := os.Hostname()
	hostname = strings.ReplaceAll(hostname, ".", "-")
	return fmt.Sprintf("termius-%s.tbk", hostname)
}
