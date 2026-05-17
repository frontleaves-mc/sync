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

	// 构建同步引擎
	engine := &SyncEngine{
		client: client,
		mcDir:  mcDir,
	}

	var errs []error

	// === Mods 同步 ===
	if err := syncMods(client, engine); err != nil {
		fail("模组同步失败: %s", err)
		errs = append(errs, err)
	}

	// === Config 同步 ===
	if err := syncConfig(client, engine); err != nil {
		fail("配置文件同步失败: %s", err)
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("同步过程中有 %d 个阶段失败", len(errs))
	}

	fmt.Println()
	fmt.Printf("  " + sGreen + sBold + "同步完成！" + sReset + "\n")
	return nil
}

func syncMods(client *SyncClient, engine *SyncEngine) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	info("正在获取服务端模组列表...")
	resp, err := client.GetModsMetadataWithMode(ctx, "server")
	if err != nil {
		return fmt.Errorf("获取模组元数据失败: %w", err)
	}

	remoteFiles := model.NormalizeModPaths(resp.Data.Files)
	info("服务端返回 %d 个模组文件", len(remoteFiles))

	diff := engine.ComputeDiff(remoteFiles, model.SyncTypeModsServer)

	printDiffSummary(diff, "模组")

	if len(diff.ToAdd) == 0 && len(diff.ToUpdate) == 0 && len(diff.ToRename) == 0 && len(diff.ToDelete) == 0 {
		fmt.Println()
		ok("所有模组已是最新")
		return nil
	}

	fmt.Println()
	info("开始同步模组...")
	result := engine.ExecuteSync(ctx, diff)

	printSyncResult(result)

	if len(result.Failed) > 0 {
		return fmt.Errorf("模组同步有 %d 个文件失败", len(result.Failed))
	}

	return nil
}

func syncConfig(client *SyncClient, engine *SyncEngine) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	info("正在获取配置文件列表...")
	resp, err := client.GetConfigMetadata(ctx)
	if err != nil {
		return fmt.Errorf("获取配置文件元数据失败: %w", err)
	}

	info("服务端返回 %d 个配置文件", len(resp.Data.Files))

	diff := engine.ComputeDiff(resp.Data.Files, model.SyncTypeConfig)

	printDiffSummary(diff, "配置文件")

	if len(diff.ToAdd) == 0 && len(diff.ToUpdate) == 0 && len(diff.ToRename) == 0 && len(diff.ToDelete) == 0 {
		fmt.Println()
		ok("所有配置文件已是最新")
		return nil
	}

	fmt.Println()
	info("开始同步配置文件...")
	result := engine.ExecuteSync(ctx, diff)

	printSyncResult(result)

	if len(result.Failed) > 0 {
		return fmt.Errorf("配置文件同步有 %d 个文件失败", len(result.Failed))
	}

	return nil
}

func printDiffSummary(diff *model.DiffResult, label string) {
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
}

func printSyncResult(result *model.SyncResult) {
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
	}
}
