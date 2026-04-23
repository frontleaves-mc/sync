package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/frontleaves-mc/sync/internal/model"
)

// SyncClient 封装与服务端同步 API 的通信。
type SyncClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewSyncClient 创建同步客户端实例。
func NewSyncClient() *SyncClient {
	return &SyncClient{
		baseURL: model.ServerBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetModsMetadata 获取服务端 mods 目录的文件元数据。
func (c *SyncClient) GetModsMetadata(ctx context.Context) (*model.SyncMetadataResponse, error) {
	return c.fetchMetadata(ctx, "/sync/mods/metadata")
}

// GetConfigMetadata 获取服务端 config 目录的文件元数据。
func (c *SyncClient) GetConfigMetadata(ctx context.Context) (*model.SyncMetadataResponse, error) {
	return c.fetchMetadata(ctx, "/sync/config/metadata")
}

// fetchMetadata 通用元数据请求。
func (c *SyncClient) fetchMetadata(ctx context.Context, path string) (*model.SyncMetadataResponse, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求服务端失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务端返回错误 (%d): %s", resp.StatusCode, string(body))
	}

	var result model.SyncMetadataResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 && result.Code != 200 {
		return nil, fmt.Errorf("服务端错误 (%d): %s", result.Code, result.Message)
	}

	return &result, nil
}

// DownloadFile 下载指定路径的文件，返回文件流和文件大小。
func (c *SyncClient) DownloadFile(ctx context.Context, relPath string) (io.ReadCloser, int64, error) {
	reqURL := c.baseURL + "/sync/download?path=" + url.QueryEscape(relPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("创建下载请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("下载请求失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, 0, fmt.Errorf("下载失败 (%d): %s", resp.StatusCode, string(body))
	}

	var size int64 = -1
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		fmt.Sscanf(contentLength, "%d", &size)
	}

	return resp.Body, size, nil
}
