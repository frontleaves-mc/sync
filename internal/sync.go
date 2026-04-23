package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/frontleaves-mc/sync/internal/model"
)

// SyncEngine 同步引擎，负责哈希对比与文件下载。
type SyncEngine struct {
	client *SyncClient
	mcDir  string
}

// NewSyncEngine 创建同步引擎实例。
func NewSyncEngine(client *SyncClient) *SyncEngine {
	// 先检查当前目录，再检查上级（二进制可能在 update/ 子目录中）
	mcDir := filepath.Join(".", model.McDirName)
	if _, err := os.Stat(mcDir); err != nil {
		mcDir = filepath.Join("..", model.McDirName)
	}
	return &SyncEngine{
		client: client,
		mcDir:  mcDir,
	}
}

// ComputeDiff 计算本地文件与服务端元数据的差异，包含重命名检测。
func (e *SyncEngine) ComputeDiff(remote []model.FileMetadata, syncType model.SyncType) *model.DiffResult {
	result := &model.DiffResult{}
	localHashes := e.scanLocalFiles(syncType)

	// 第一步：按路径匹配
	for _, rf := range remote {
		if localHash, exists := localHashes[rf.Path]; exists {
			if localHash == rf.Hash {
				result.Unchanged++
			} else {
				result.ToUpdate = append(result.ToUpdate, rf)
			}
			delete(localHashes, rf.Path)
		} else {
			result.ToAdd = append(result.ToAdd, rf)
		}
	}

	// 第二步：对未匹配的远程文件，通过 hash 检测重命名
	hashToPath := make(map[string]string)
	for path, hash := range localHashes {
		if _, exists := hashToPath[hash]; !exists {
			hashToPath[hash] = path
		}
	}

	var toAdd []model.FileMetadata
	for _, rf := range result.ToAdd {
		if localPath, found := hashToPath[rf.Hash]; found {
			result.ToRename = append(result.ToRename, model.RenameEntry{
				OldPath: localPath,
				NewPath: rf.Path,
			})
			delete(hashToPath, rf.Hash)
		} else {
			toAdd = append(toAdd, rf)
		}
	}
	result.ToAdd = toAdd

	return result
}

// scanLocalFiles 扫描本地目录，返回 path→hash 映射。
func (e *SyncEngine) scanLocalFiles(syncType model.SyncType) map[string]string {
	hashes := make(map[string]string)

	var dir string
	switch syncType {
	case model.SyncTypeMods:
		dir = filepath.Join(e.mcDir, "mods")
	case model.SyncTypeConfig:
		dir = filepath.Join(e.mcDir, "config")
	default:
		return hashes
	}

	e.scanDirRecursive(dir, string(syncType), syncType == model.SyncTypeConfig, hashes)
	return hashes
}

// scanDirRecursive 递归扫描目录计算文件哈希。
func (e *SyncEngine) scanDirRecursive(dirPath, prefix string, recursive bool, hashes map[string]string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if recursive {
				e.scanDirRecursive(
					filepath.Join(dirPath, entry.Name()),
					prefix+"/"+entry.Name(),
					true,
					hashes,
				)
			}
			continue
		}

		if !recursive && !strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			continue
		}

		fullPath := filepath.Join(dirPath, entry.Name())
		hash, err := e.computeLocalHash(fullPath)
		if err != nil {
			continue
		}

		relPath := prefix + "/" + entry.Name()
		hashes[relPath] = "sha256:" + hash
	}
}

// computeLocalHash 计算本地文件的 SHA-256 哈希。
func (e *SyncEngine) computeLocalHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// DownloadAndVerify 下载文件并校验哈希，成功返回 nil。
func (e *SyncEngine) DownloadAndVerify(ctx context.Context, meta model.FileMetadata) error {
	targetPath := filepath.Join(e.mcDir, meta.Path)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	tmpPath := targetPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer f.Close()

	stream, _, err := e.client.DownloadFile(ctx, meta.Path)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("下载失败: %w", err)
	}
	defer stream.Close()

	h := sha256.New()
	mw := io.MultiWriter(f, h)

	if _, err := io.Copy(mw, stream); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("写入文件失败: %w", err)
	}

	actualHash := "sha256:" + hex.EncodeToString(h.Sum(nil))
	if actualHash != meta.Hash {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("哈希校验失败: 期望 %s, 实际 %s", meta.Hash, actualHash)
	}

	// Windows 不允许重命名有打开句柄的文件，必须在 Rename 前显式关闭
	f.Close()

	if err := os.Rename(tmpPath, targetPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	return nil
}

// RenameFile 将本地文件从旧路径重命名为新路径。
func (e *SyncEngine) RenameFile(oldRelPath, newRelPath string) error {
	oldPath := filepath.Join(e.mcDir, oldRelPath)
	newPath := filepath.Join(e.mcDir, newRelPath)

	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	return nil
}

// ExecuteSync 执行完整的同步操作。
func (e *SyncEngine) ExecuteSync(ctx context.Context, diff *model.DiffResult) *model.SyncResult {
	result := &model.SyncResult{}
	var mu sync.Mutex

	// 先执行重命名（快速，无网络 IO）
	for _, entry := range diff.ToRename {
		if err := e.RenameFile(entry.OldPath, entry.NewPath); err != nil {
			mu.Lock()
			result.Failed = append(result.Failed, model.FailedFile{
				Path:   entry.NewPath,
				Reason: err.Error(),
			})
			mu.Unlock()
		} else {
			mu.Lock()
			result.Renamed++
			mu.Unlock()
		}
	}

	// 再并发下载新增和更新的文件
	downloadFiles := append(diff.ToAdd, diff.ToUpdate...)

	var wg sync.WaitGroup
	sem := make(chan struct{}, model.MaxDownloadWorkers)

	for _, file := range downloadFiles {
		wg.Add(1)
		sem <- struct{}{}

		go func(meta model.FileMetadata) {
			defer wg.Done()
			defer func() { <-sem }()

			err := e.DownloadAndVerify(ctx, meta)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.Failed = append(result.Failed, model.FailedFile{
					Path:   meta.Path,
					Reason: err.Error(),
				})
			} else {
				result.Downloaded++
			}
		}(file)
	}

	wg.Wait()

	return result
}
