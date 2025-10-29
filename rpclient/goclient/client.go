package goclient

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	spdk "github.com/spdk/spdk/go/rpc/client"
)

const SocketAddress = "/var/tmp/spdk.sock"

var (
	clientInstance *spdk.Client
	once           sync.Once
)

// GetClient 获取单例 RPC 客户端
func GetClient() *spdk.Client {
	once.Do(func() {
		c, err := spdk.CreateClientWithJsonCodec(spdk.Unix, SocketAddress)
		if err != nil {
			log.Fatalf("无法连接  RPC (%s):", SocketAddress)
		}
		clientInstance = c
	})
	return clientInstance
}

// Call 通用 RPC 调用方法
func Call(method string, params map[string]any) ([]byte, error) {
	client := GetClient()
	resp, err := client.Call(method, params)
	if err != nil {
		return nil, fmt.Errorf("RPC 调用失败 (%s): %w", method, err)
	}

	data, err := json.MarshalIndent(resp.Result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化失败: %w", err)
	}
	return data, nil
}

// BuildParams 构建 RPC 参数（自动过滤空值）
func BuildParams(args map[string]any) map[string]any {
	params := make(map[string]any)
	for k, v := range args {
		if v == nil {
			continue
		}
		switch val := v.(type) {
		case string:
			if val != "" {
				params[k] = val
			}
		default:
			params[k] = val
		}
	}
	return params
}

// Close 在 CLI 退出时关闭
func Close() {
	if clientInstance != nil {
		clientInstance.Close()
	}
}
