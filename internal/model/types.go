package model

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	AppName    = "FrontLeaves 模组同步器"
	AppVersion = "v1.0.0"

	ServerBaseURL = "https://yggleaf.frontleaves.com/api/v1"
	McDirName     = ".minecraft"

	MaxDownloadWorkers = 4
)

// SyncType 同步类型。
type SyncType string

const (
	SyncTypeMods   SyncType = "mods"
	SyncTypeConfig SyncType = "config"
)

// FileMetadata 服务端返回的文件元数据。
type FileMetadata struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// SyncMetadataResponse 元数据 API 响应。
type SyncMetadataResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Files     []FileMetadata `json:"files"`
		Total     int            `json:"total"`
		ScannedAt string         `json:"scanned_at"`
	} `json:"data"`
}

// DiffResult 差异计算结果。
type DiffResult struct {
	ToAdd     []FileMetadata
	ToUpdate  []FileMetadata
	ToRename  []RenameEntry
	Unchanged int
}

// RenameEntry 重命名条目，记录旧路径到新路径的映射。
type RenameEntry struct {
	OldPath string
	NewPath string
}

// SyncResult 同步执行结果。
type SyncResult struct {
	Downloaded int
	Renamed    int
	Failed     []FailedFile
}

// FailedFile 失败文件记录。
type FailedFile struct {
	Path   string
	Reason string
}

// FormatSize 格式化文件大小为人类可读形式。
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// CheckResult 环境检查结果。
type CheckResult struct {
	McDirFound    bool
	ModsDirOK     bool
	ConfigDirOK   bool
	McDirPath     string
	ModsDirPath   string
	ConfigDirPath string
}

// CheckEnvironment 检查运行环境，验证 .minecraft 目录和子目录。
func CheckEnvironment() CheckResult {
	// 二进制在 update/ 子目录中运行，先检查当前目录，再检查上级
	mcPath := filepath.Join(".", McDirName)
	if _, err := os.Stat(mcPath); err != nil {
		mcPath = filepath.Join("..", McDirName)
	}

	result := CheckResult{
		McDirPath:     mcPath,
		ModsDirPath:   filepath.Join(mcPath, "mods"),
		ConfigDirPath: filepath.Join(mcPath, "config"),
	}

	info, err := os.Stat(mcPath)
	if err != nil || !info.IsDir() {
		return result
	}
	result.McDirFound = true

	if info, err := os.Stat(result.ModsDirPath); err == nil && info.IsDir() {
		result.ModsDirOK = true
	}
	if info, err := os.Stat(result.ConfigDirPath); err == nil && info.IsDir() {
		result.ConfigDirOK = true
	}

	return result
}
