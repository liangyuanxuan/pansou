package elasticsearch

import (
	_ "context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"pansou/config"

	"github.com/elastic/go-elasticsearch/v8"
)

var (
	// ESClient 全局ES客户端实例
	ESClient *elasticsearch.Client
)

// InitESClient 初始化Elasticsearch客户端
func InitESClient(cfg *config.Config) error {
	// 如果未启用ES，跳过初始化
	if !cfg.ESEnabled {
		return nil
	}

	// 配置ES客户端
	esCfg := elasticsearch.Config{
		Addresses: cfg.ESAddresses,
		Username:  cfg.ESUsername,
		Password:  cfg.ESPassword,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 根据需要配置
			},
		},
	}

	// 创建客户端
	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return fmt.Errorf("创建ES客户端失败: %w", err)
	}

	// 测试连接（不使用context，使用默认配置）
	res, err := client.Info()
	if err != nil {
		return fmt.Errorf("ES连接测试失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ES响应错误: %s", res.Status())
	}

	ESClient = client
	return nil
}

// GetClient 获取ES客户端
func GetClient() *elasticsearch.Client {
	return ESClient
}

// IsEnabled 检查ES是否启用
func IsEnabled() bool {
	return ESClient != nil
}
