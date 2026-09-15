package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	Version   = "1.0.0"
	BackupExt = ".tbk"
)

type Metadata struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Hostname  string    `json:"hostname"`
	LocalKey  string    `json:"local_key"`
}

type FileEntry struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type BackupInfo struct {
	Metadata Metadata    `json:"metadata"`
	Files    []FileEntry `json:"files"`
}

type BackupResult struct {
	Path      string `json:"path"`
	FileCount int    `json:"fileCount"`
}

type RestoreResult struct {
	Restored int `json:"restored"`
}

type Platform interface {
	DataDir() (string, error)
	GetKey() (string, error)
	SetKey(key string) error
	IsTermiusRunning() bool
	CloseHint() string
	OSName() string
}

func Backup(p Platform, outPath string, forceIfRunning bool) (*BackupResult, error) {
	if p.IsTermiusRunning() && !forceIfRunning {
		return nil, fmt.Errorf("termius_running")
	}

	dataDir, err := p.DataDir()
	if err != nil {
		return nil, err
	}
	if err := checkDataDir(dataDir); err != nil {
		return nil, err
	}

	localKey, err := p.GetKey()
	if err != nil {
		return nil, err
	}

	if outPath == "" {
		hostname, _ := os.Hostname()
		hostname = strings.ReplaceAll(hostname, ".", "-")
		outPath = fmt.Sprintf("termius-%s-%s%s", hostname, time.Now().Format("2006-01-02"), BackupExt)
	} else if !strings.HasSuffix(outPath, BackupExt) {
		outPath += BackupExt
	}

	f, err := os.Create(outPath)
	if err != nil {
		return nil, fmt.Errorf("无法创建备份文件：%w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	hostname, _ := os.Hostname()
	meta := Metadata{
		Version:   Version,
		CreatedAt: time.Now(),
		Hostname:  hostname,
		LocalKey:  localKey,
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	if err := writeToTar(tw, "metadata.json", metaJSON); err != nil {
		return nil, err
	}

	dirs := []string{
		filepath.Join("Local Storage", "leveldb"),
		filepath.Join("IndexedDB", "file__0.indexeddb.leveldb"),
	}

	fileCount := 0
	for _, dir := range dirs {
		fullDir := filepath.Join(dataDir, dir)
		if _, err := os.Stat(fullDir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(fullDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "LOCK" {
				continue
			}
			filePath := filepath.Join(fullDir, entry.Name())
			tarPath := filepath.Join(dir, entry.Name())

			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			if err := writeToTar(tw, tarPath, data); err != nil {
				return nil, err
			}
			fileCount++
		}
	}

	for _, extra := range []string{"window-state.json", "Preferences"} {
		fp := filepath.Join(dataDir, extra)
		data, err := os.ReadFile(fp)
		if err == nil {
			if err := writeToTar(tw, extra, data); err != nil {
				return nil, err
			}
			fileCount++
		}
	}

	return &BackupResult{Path: outPath, FileCount: fileCount}, nil
}

func Restore(p Platform, inPath string) (*RestoreResult, error) {
	if p.IsTermiusRunning() {
		return nil, fmt.Errorf("Termius 正在运行，恢复前请先关闭它。\n%s", p.CloseHint())
	}

	if _, err := os.Stat(inPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("未找到备份文件：%s", inPath)
	}

	dataDir, err := p.DataDir()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(inPath)
	if err != nil {
		return nil, fmt.Errorf("无法打开备份文件：%w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("无效的备份文件（非 gzip 格式）：%w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	var meta Metadata
	files := make(map[string][]byte)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取备份时出错：%w", err)
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("从备份中读取 %s 时出错：%w", header.Name, err)
		}

		if header.Name == "metadata.json" {
			if err := json.Unmarshal(data, &meta); err != nil {
				return nil, fmt.Errorf("备份中的元数据无效：%w", err)
			}
		} else {
			files[header.Name] = data
		}
	}

	if meta.Version == "" {
		return nil, fmt.Errorf("无效的备份文件：未找到元数据")
	}

	if meta.LocalKey != "" {
		if err := p.SetKey(meta.LocalKey); err != nil {
			return nil, fmt.Errorf("无法恢复加密密钥：%w", err)
		}
	}

	restored := 0
	for name, data := range files {
		destPath := filepath.Join(dataDir, name)

		if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
			continue
		}

		if err := os.WriteFile(destPath, data, 0600); err != nil {
			continue
		}
		restored++
	}

	return &RestoreResult{Restored: restored}, nil
}

func Inspect(inPath string) (*BackupInfo, error) {
	f, err := os.Open(inPath)
	if err != nil {
		return nil, fmt.Errorf("无法打开备份文件：%w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("无效的备份文件：%w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	info := &BackupInfo{}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取备份时出错：%w", err)
		}

		data, _ := io.ReadAll(tr)

		if header.Name == "metadata.json" {
			json.Unmarshal(data, &info.Metadata)
		} else {
			info.Files = append(info.Files, FileEntry{
				Name: header.Name,
				Size: len(data),
			})
		}
	}

	if info.Metadata.Version == "" {
		return nil, fmt.Errorf("无效的备份文件：未找到元数据")
	}

	return info, nil
}

func checkDataDir(dataDir string) error {
	_, err := os.Stat(dataDir)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return fmt.Errorf("未找到 Termius 数据目录：%s", dataDir)
	}
	if isAccessDenied(err) {
		return fmt.Errorf("无法访问 Termius 数据目录：%s\n请在「系统设置 → 隐私与安全性 → 完全磁盘访问权限」中允许 Termius Tools", dataDir)
	}
	return fmt.Errorf("无法访问 Termius 数据目录：%s：%w", dataDir, err)
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

func writeToTar(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name:    name,
		Size:    int64(len(data)),
		Mode:    0600,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("无法为 %s 写入 tar 头：%w", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("无法为 %s 写入 tar 数据：%w", name, err)
	}
	return nil
}
