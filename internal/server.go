package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/frontleaves-mc/sync/internal/model"
)

const (
	sRed    = "\033[0;31m"
	sGreen  = "\033[0;32m"
	sYellow = "\033[0;33m"
	sCyan   = "\033[0;36m"
	sBold   = "\033[1m"
	sReset  = "\033[0m"
)

func info(format string, args ...any) {
	fmt.Printf(sCyan+sBold+"[INFO]"+sReset+" "+format+"\n", args...)
}

func ok(format string, args ...any) {
	fmt.Printf(sGreen+sBold+"[OK]"+sReset+" "+format+"\n", args...)
}

func fail(format string, args ...any) {
	fmt.Printf(sRed+sBold+"[FAIL]"+sReset+" "+format+"\n", args...)
}

// RunServerSync 纯 CLI 模式的服务端模组同步入口。
func RunServerSync() error {
	// Banner
	fmt.Println()
	fmt.Printf("  "+sCyan+sBold+"%s %s"+sReset+"\n", model.AppName, model.AppVersion)
	fmt.Println()

	// 检测 mods/ 目录
	mcDir := ""
	modsDir := ""

	if info, err := os.Stat("./mods"); err == nil && info.IsDir() {
		modsDir = "./mods"
		mcDir = "."
	} else if info, err := os.Stat("../mods"); err == nil && info.IsDir() {
		modsDir = "../mods"
		mcDir = ".."
	}

	if modsDir == "" {
		info("未找到 mods/ 目录，自动创建 ./mods/")
		if err := os.MkdirAll("./mods", 0o755); err != nil {
			return fmt.Errorf("创建 mods 目录失败: %w", err)
		}
		modsDir = "./mods"
		mcDir = "."
	}

	absMods, _ := filepath.Abs(modsDir)
	info("模组目录: %s", absMods)

	// 创建客户端
	client := NewSyncClient()

	// 获取元数据
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	info("正在获取服务端模组列表...")
	resp, err := client.GetModsMetadataWithMode(ctx, "server")
	if err != nil {
		return fmt.Errorf("获取模组元数据失败: %w", err)
	}

	remoteFiles := model.NormalizeModPaths(resp.Data.Files)
	info("服务端返回 %d 个模组文件", len(remoteFiles))

	// 构建同步引擎（直接构造，避免 NewSyncEngine 的 .minecraft 检测）
	engine := &SyncEngine{
		client: client,
		mcDir:  mcDir,
	}

	// 计算差异
	diff := engine.ComputeDiff(remoteFiles, model.SyncTypeModsServer)

	// 打印差异摘要
	fmt.Println()
	if diff.Unchanged > 0 {
		fmt.Printf("  "+sGreen+"■ 未变更: %d 个文件"+sReset+"\n", diff.Unchanged)
	}
	if len(diff.ToAdd) > 0 {
		fmt.Printf("  "+sYellow+"■ 新增: %d 个文件"+sReset+"\n", len(diff.ToAdd))
	}
	if len(diff.ToUpdate) > 0 {
		fmt.Printf("  "+sYellow+"■ 更新: %d 个文件"+sReset+"\n", len(diff.ToUpdate))
	}
	if len(diff.ToRename) > 0 {
		fmt.Printf("  "+sCyan+"■ 重命名: %d 个文件"+sReset+"\n", len(diff.ToRename))
	}
	if len(diff.ToDelete) > 0 {
		fmt.Printf("  "+sRed+"■ 删除: %d 个文件"+sReset+"\n", len(diff.ToDelete))
	}

	if len(diff.ToAdd) == 0 && len(diff.ToUpdate) == 0 && len(diff.ToRename) == 0 && len(diff.ToDelete) == 0 {
		fmt.Println()
		ok(sGreen + sBold + "所有模组已是最新" + sReset)
		return nil
	}

	// 执行同步
	fmt.Println()
	info("开始同步...")
	result := engine.ExecuteSync(ctx, diff)

	// 打印结果
	fmt.Println()
	if result.Downloaded > 0 {
		ok("下载: %d 个", result.Downloaded)
	}
	if result.Renamed > 0 {
		ok("重命名: %d 个", result.Renamed)
	}
	if result.Deleted > 0 {
		ok("删除: %d 个", result.Deleted)
	}

	if len(result.Failed) > 0 {
		fail("失败: %d 个", len(result.Failed))
		for _, f := range result.Failed {
			fmt.Printf("    "+sRed+"• %s: %s"+sReset+"\n", f.Path, f.Reason)
		}
		return fmt.Errorf("同步过程中有 %d 个文件失败", len(result.Failed))
	}

	fmt.Println()
	fmt.Printf("  " + sGreen + sBold + "同步完成！" + sReset + "\n")
	return nil
}
