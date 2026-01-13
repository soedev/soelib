package soehttp

import (
	"crypto/md5"
	"encoding/hex"
	"sync"

	"github.com/afex/hystrix-go/hystrix"
)

// HystrixConfig 熔断配置（实例级）
type HystrixConfig struct {
	Timeout                int // 超时时间（毫秒）
	MaxConcurrentRequests  int // 最大并发请求数
	ErrorPercentThreshold  int // 错误率阈值（百分比）
	RequestVolumeThreshold int // 触发熔断的最小请求数
	SleepWindow            int // 熔断恢复时间窗口（毫秒）
}

// DefaultHystrixConfig 返回默认熔断配置（适合微服务内部调用）
func DefaultHystrixConfig() *HystrixConfig {
	return &HystrixConfig{
		Timeout:                2000, // 2秒
		MaxConcurrentRequests:  100,
		ErrorPercentThreshold:  50,   // 50%错误率
		RequestVolumeThreshold: 20,   // 至少20个请求
		SleepWindow:            5000, // 5秒恢复窗口
	}
}

// StrictHystrixConfig 返回严格的熔断配置（适合关键业务）
func StrictHystrixConfig() *HystrixConfig {
	return &HystrixConfig{
		Timeout:                1000, // 1秒
		MaxConcurrentRequests:  50,
		ErrorPercentThreshold:  30,   // 30%错误率（更严格）
		RequestVolumeThreshold: 10,   // 更敏感
		SleepWindow:            3000, // 3秒快速恢复
	}
}

// RelaxedHystrixConfig 返回宽松的熔断配置（适合外部服务）
func RelaxedHystrixConfig() *HystrixConfig {
	return &HystrixConfig{
		Timeout:                5000, // 5秒（外部服务可能更慢）
		MaxConcurrentRequests:  200,
		ErrorPercentThreshold:  60, // 60%错误率（更宽松）
		RequestVolumeThreshold: 30,
		SleepWindow:            10000, // 10秒恢复窗口
	}
}

var (
	commandConfigMu sync.RWMutex
	configuredCmds  = make(map[string]bool) // 记录已配置的命令
)

// generateCommandName 为每个实例生成唯一的熔断器命令名
// 使用 URL 的哈希值确保同一 URL 的不同实例使用相同的熔断器
func generateCommandName(url string) string {
	hash := md5.Sum([]byte(url))
	return "soehttp-" + hex.EncodeToString(hash[:])[:16]
}

// configureHystrixCommand 配置熔断器命令
func configureHystrixCommand(name string, config *HystrixConfig) {
	commandConfigMu.Lock()
	defer commandConfigMu.Unlock()

	// 避免重复配置
	if configuredCmds[name] {
		return
	}

	hystrix.ConfigureCommand(name, hystrix.CommandConfig{
		Timeout:                config.Timeout,
		MaxConcurrentRequests:  config.MaxConcurrentRequests,
		ErrorPercentThreshold:  config.ErrorPercentThreshold,
		RequestVolumeThreshold: config.RequestVolumeThreshold,
		SleepWindow:            config.SleepWindow,
	})

	configuredCmds[name] = true
}

// ResetHystrixCommands 重置所有熔断器命令配置（用于测试）
func ResetHystrixCommands() {
	commandConfigMu.Lock()
	defer commandConfigMu.Unlock()

	hystrix.Flush()
	configuredCmds = make(map[string]bool)
}
