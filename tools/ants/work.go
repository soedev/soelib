package ants

import (
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/panjf2000/ants/v2"
)

var (
	once     sync.Once
	antsPool *Pool
)

// PoolConfig 全局池配置
var PoolConfig = struct {
	Name     string
	Size     int
	PreAlloc bool
}{
	Name:     "main-pool",
	Size:     runtime.NumCPU() * 4,
	PreAlloc: false,
}

// InitCoroutinePool 初始化全局协程池，使用默认配置
func InitCoroutinePool() {
	InitCoroutinePoolWithConfig(PoolConfig.Name, PoolConfig.Size, PoolConfig.PreAlloc)
}

// InitCoroutinePoolWithConfig 初始化全局协程池，使用自定义配置
func InitCoroutinePoolWithConfig(name string, size int, preAlloc bool) {
	once.Do(func() {
		// 创建协程池，使用预分配配置
		var err error
		antsPool, err = NewWithOptions(name, size, func(opts *ants.Options) {
			opts.PreAlloc = preAlloc
		})
		if err != nil {
			log.Printf("[ants][%s] 初始化协程池失败: %v", name, err)
			// 初始化失败时，使用默认配置重试
			antsPool, _ = New(name)
		} else {
			log.Printf("[ants][%s] 协程池初始化成功，大小: %d, 预分配: %t", name, size, preAlloc)
		}
	})
}

// CoroutineRelease 释放全局协程池资源
func CoroutineRelease() {
	if antsPool != nil {
		err := antsPool.Release()
		if err != nil {
			log.Printf("[ants][%s] 优雅关闭，变得不优雅了: %v", antsPool.name, err)
		} else {
			log.Printf("[ants][%s] 协程池已优雅关闭", antsPool.name)
		}
	}
}

// SubmitTask 向全局协程池提交任务
func SubmitTask(task func()) {
	if antsPool == nil {
		log.Printf("[ants] 协程池未初始化，无法提交任务")
		return
	}

	err := antsPool.Submit(task)
	if err != nil {
		log.Printf(fmt.Sprintf("[ants][%s] SubmitTask，发生异常: %v", antsPool.name, err))
	}
}

// GetPool 获取全局协程池实例
func GetPool() *Pool {
	return antsPool
}

// NewPool 创建一个新的协程池实例，提供便捷的工厂方法
func NewPool(name string, size int, options ...func(*ants.Options)) (*Pool, error) {
	return NewWithOptions(name, size, options...)
}
