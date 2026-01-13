package ants

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
)

var (
	once           sync.Once
	antsPool       *Pool
	monitorTicker  *time.Ticker
	monitorStopCh  chan struct{}
	monitorRunning bool
	monitorMu      sync.Mutex
)

// HealthConfig 健康分析配置
type HealthConfig struct {
	Enabled              bool          // 是否启用健康分析（针对带ID的任务）
	SlowTaskThreshold    time.Duration // 慢任务阈值，0表示不判断慢任务
	MaxSlowTaskRecords   int           // 最大慢任务记录数（默认100）
	MaxFailedTaskRecords int           // 最大异常任务记录数（默认100）
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Enabled             bool                  // 是否启用自动监控打印
	Interval            time.Duration         // 监控打印间隔（默认5分钟）
	PrintBasicMetrics   bool                  // 是否打印基础指标
	PrintHealthInfo     bool                  // 是否打印健康分析信息
	MaxSlowTasksPrint   int                   // 最多打印多少个慢任务详情（默认5）
	MaxFailedTasksPrint int                   // 最多打印多少个异常任务详情（默认5）
	CustomPrinter       func(metrics Metrics) // 自定义打印函数（可选）
}

// DefaultHealthConfig 返回默认健康分析配置
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		Enabled:              false, // 默认关闭，不影响性能
		SlowTaskThreshold:    0,     // 0表示不判断慢任务
		MaxSlowTaskRecords:   100,
		MaxFailedTaskRecords: 100,
	}
}

// DefaultMonitorConfig 返回默认监控配置
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		Enabled:             false,           // 默认关闭
		Interval:            time.Minute * 5, // 默认5分钟
		PrintBasicMetrics:   true,
		PrintHealthInfo:     true,
		MaxSlowTasksPrint:   5,
		MaxFailedTasksPrint: 5,
		CustomPrinter:       nil,
	}
}

// PoolConfig 全局池配置
var PoolConfig = struct {
	Name     string
	Size     int
	PreAlloc bool
	Health   HealthConfig // 健康分析配置
}{
	Name:     "main-pool",
	Size:     runtime.NumCPU() * 200,
	PreAlloc: true,
	Health:   DefaultHealthConfig(),
}

// InitCoroutinePool 初始化全局协程池（默认配置）
func InitCoroutinePool() {
	InitCoroutinePoolWithConfig(GlobalPoolConfig{
		Name:     PoolConfig.Name,
		Size:     PoolConfig.Size,
		PreAlloc: PoolConfig.PreAlloc,
		Health:   PoolConfig.Health,
	})
}

// GlobalPoolConfig 全局池完整配置
type GlobalPoolConfig struct {
	Name     string        // 线程池名称
	Size     int           // 线程池大小
	PreAlloc bool          // 是否预分配
	Health   HealthConfig  // 健康分析配置
	Monitor  MonitorConfig // 监控配置
}

// InitCoroutinePoolWithConfig 初始化全局协程池(自定义配置)
func InitCoroutinePoolWithConfig(config GlobalPoolConfig) {
	once.Do(func() {
		// 创建协程池，使用预分配配置
		var err error
		antsPool, err = NewWithOptions(config.Name, config.Size, func(opts *ants.Options) {
			opts.PreAlloc = config.PreAlloc
		})
		if err != nil {
			log.Printf("[ants][%s] 初始化协程池失败: %v", config.Name, err)
			// 初始化失败时，使用默认配置重试
			antsPool, _ = New(config.Name)
		} else {
			// 设置健康分析配置
			antsPool.SetHealthConfig(config.Health)
			log.Printf("[ants][%s] 协程池初始化成功，大小: %d, 预分配: %t, 健康分析: %t",
				config.Name, config.Size, config.PreAlloc, config.Health.Enabled)

			// 启动自动监控（如果启用）
			if config.Monitor.Enabled {
				StartGlobalMonitor(config.Monitor)
			}
		}
	})
}

// CoroutineRelease 释放全局协程池资源
func CoroutineRelease() {
	// 先停止监控
	StopGlobalMonitor()

	if antsPool != nil {
		err := antsPool.Release()
		if err != nil {
			log.Printf("[ants][%s] 优雅关闭，变得不优雅了: %v", antsPool.name, err)
		} else {
			log.Printf("[ants][%s] 协程池已优雅关闭", antsPool.name)
		}
	}
}

// SubmitTask 向全局协程池提交任务（不追踪）
//
// ⚠️ 并发安全注意事项：
//
// 1. 避免在闭包中直接引用外部的 map、slice、指针等可变数据
// 2. 如需传递数据，请在提交前拷贝或使用 SubmitTaskWithData
// 3. 字符串、数字等值类型可以安全使用
//
// 示例（不安全）：
//
//	data := map[string]string{"key": "value"}
//	ants.SubmitTask(func() {
//	    json.Marshal(data) // ❌ 可能并发访问 data
//	})
//
// 示例（安全方式1 - 手动序列化）：
//
//	data := map[string]string{"key": "value"}
//	dataBytes, _ := json.Marshal(data) // ✅ 先序列化
//	ants.SubmitTask(func() {
//	    var d map[string]string
//	    json.Unmarshal(dataBytes, &d) // ✅ 使用副本
//	})
//
// 示例（安全方式2 - 使用辅助方法）：
//
//	data := map[string]string{"key": "value"}
//	ants.SubmitTaskWithData(data, func(d interface{}) {
//	    // d 是安全的副本
//	})
func SubmitTask(task func()) {
	if antsPool == nil {
		log.Printf("[ants] 协程池未初始化，无法提交任务")
		return
	}

	err := antsPool.Submit(task)
	if err != nil {
		log.Printf("[ants][%s] SubmitTask，发生异常: %v", antsPool.name, err)
	}
}

// SubmitTaskWithID 向全局协程池提交带任务ID的任务（会被健康分析追踪）
func SubmitTaskWithID(taskID string, task func()) {
	if antsPool == nil {
		log.Printf("[ants] 协程池未初始化，无法提交任务")
		return
	}

	err := antsPool.SubmitWithTaskID(taskID, task)
	if err != nil {
		log.Printf("[ants][%s] SubmitTaskWithID[%s]，发生异常: %v", antsPool.name, taskID, err)
	}
}

// SubmitTaskWithData 向全局协程池提交带数据的任务（自动处理数据拷贝，确保并发安全）
//
// 此方法通过 JSON 序列化/反序列化自动创建数据副本，避免并发访问问题。
// 适用于数据可以被 JSON 序列化的场景（struct、map、slice 等）。
//
// 参数：
//   - data: 要传递给任务的数据（会被自动拷贝）
//   - task: 任务函数，接收拷贝后的数据
//
// 示例：
//
//	// 传递 map
//	data := map[string]interface{}{
//	    "userId": 123,
//	    "action": "login",
//	}
//	ants.SubmitTaskWithData(data, func(d interface{}) {
//	    dataMap := d.(map[string]interface{})
//	    fmt.Println(dataMap["userId"])
//	})
//
//	// 传递 struct
//	user := User{ID: 123, Name: "test"}
//	ants.SubmitTaskWithData(user, func(d interface{}) {
//	    u := d.(User)
//	    fmt.Println(u.Name)
//	})
//
// 注意：
//   - 数据必须可以被 JSON 序列化
//   - 不支持 channel、func、循环引用等类型
//   - 有序列化开销，但比锁竞争更高效
func SubmitTaskWithData(data interface{}, task func(interface{})) error {
	if antsPool == nil {
		err := fmt.Errorf("协程池未初始化")
		log.Printf("[ants] %v", err)
		return err
	}

	// 在主 goroutine 中序列化数据
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("[ants][%s] SubmitTaskWithData 序列化数据失败: %v", antsPool.name, err)
		return fmt.Errorf("序列化数据失败: %w", err)
	}

	// 提交到协程池
	err = antsPool.Submit(func() {
		// 在协程池中反序列化，得到独立副本
		var clonedData interface{}
		if err := json.Unmarshal(dataBytes, &clonedData); err != nil {
			log.Printf("[ants][%s] SubmitTaskWithData 反序列化数据失败: %v", antsPool.name, err)
			return
		}

		// 执行任务，使用副本
		task(clonedData)
	})

	if err != nil {
		log.Printf("[ants][%s] SubmitTaskWithData 提交任务失败: %v", antsPool.name, err)
		return fmt.Errorf("提交任务失败: %w", err)
	}

	return nil
}

// SubmitTaskGeneric 向全局协程池提交带类型安全数据的任务（泛型版本，自动拷贝）
//
// 此方法使用泛型提供类型安全的数据传递，通过 JSON 序列化/反序列化创建数据副本。
// 需要 Go 1.18 或更高版本。
//
// 参数：
//   - data: 要传递给任务的数据（会被自动拷贝）
//   - task: 任务函数，接收拷贝后的数据（类型安全）
//
// 示例：
//
//	// 传递 struct（类型安全）
//	type User struct {
//	    ID   int
//	    Name string
//	}
//	user := User{ID: 123, Name: "test"}
//	ants.SubmitTaskGeneric(user, func(u User) {
//	    fmt.Println(u.Name) // 类型安全，无需类型断言
//	})
//
//	// 传递 map
//	data := map[string]string{"key": "value"}
//	ants.SubmitTaskGeneric(data, func(d map[string]string) {
//	    fmt.Println(d["key"]) // 类型安全
//	})
//
// 优点：
//   - 类型安全，编译时检查
//   - 自动拷贝，并发安全
//   - API 简洁，无需类型断言
//
// 注意：
//   - 数据必须可以被 JSON 序列化
//   - 需要 Go 1.18+
func SubmitTaskGeneric[T any](data T, task func(T)) error {
	if antsPool == nil {
		err := fmt.Errorf("协程池未初始化")
		log.Printf("[ants] %v", err)
		return err
	}

	// 在主 goroutine 中序列化数据
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("[ants][%s] SubmitTaskGeneric 序列化数据失败: %v", antsPool.name, err)
		return fmt.Errorf("序列化数据失败: %w", err)
	}

	// 提交到协程池
	err = antsPool.Submit(func() {
		// 在协程池中反序列化，得到独立副本
		var clonedData T
		if err := json.Unmarshal(dataBytes, &clonedData); err != nil {
			log.Printf("[ants][%s] SubmitTaskGeneric 反序列化数据失败: %v", antsPool.name, err)
			return
		}

		// 执行任务，使用副本
		task(clonedData)
	})

	if err != nil {
		log.Printf("[ants][%s] SubmitTaskGeneric 提交任务失败: %v", antsPool.name, err)
		return fmt.Errorf("提交任务失败: %w", err)
	}

	return nil
}

// SetGlobalHealthConfig 设置全局协程池的健康分析配置
func SetGlobalHealthConfig(config HealthConfig) {
	if antsPool == nil {
		log.Printf("[ants] 协程池未初始化，无法设置健康分析配置")
		return
	}
	antsPool.SetHealthConfig(config)
	log.Printf("[ants][%s] 健康分析配置已更新，启用: %t, 慢任务阈值: %v",
		antsPool.name, config.Enabled, config.SlowTaskThreshold)
}

// GetGlobalHealthConfig 获取全局协程池的健康分析配置
func GetGlobalHealthConfig() HealthConfig {
	if antsPool == nil {
		log.Printf("[ants] 协程池未初始化")
		return DefaultHealthConfig()
	}
	return antsPool.GetHealthConfig()
}

// GetGlobalMetrics 获取全局协程池的监控指标
func GetGlobalMetrics() Metrics {
	if antsPool == nil {
		log.Printf("[ants] 协程池未初始化")
		return Metrics{}
	}
	return antsPool.Metrics()
}

// GetPool 获取全局协程池实例
func GetPool() *Pool {
	return antsPool
}

// NewPool 创建一个新的协程池实例，提供便捷的工厂方法
func NewPool(name string, size int, options ...func(*ants.Options)) (*Pool, error) {
	return NewWithOptions(name, size, options...)
}

// StartGlobalMonitor 启动全局协程池监控
func StartGlobalMonitor(config MonitorConfig) {
	monitorMu.Lock()
	defer monitorMu.Unlock()

	// 如果已经在运行，先停止
	if monitorRunning {
		StopGlobalMonitor()
	}

	if antsPool == nil {
		log.Printf("[ants] 协程池未初始化，无法启动监控")
		return
	}

	// 设置默认值
	if config.Interval <= 0 {
		config.Interval = time.Minute * 5
	}
	if config.MaxSlowTasksPrint <= 0 {
		config.MaxSlowTasksPrint = 5
	}
	if config.MaxFailedTasksPrint <= 0 {
		config.MaxFailedTasksPrint = 5
	}

	monitorStopCh = make(chan struct{})
	monitorTicker = time.NewTicker(config.Interval)
	monitorRunning = true

	log.Printf("[ants][%s] 监控已启动，间隔: %v", antsPool.name, config.Interval)

	go func() {
		for {
			select {
			case <-monitorTicker.C:
				printMonitorMetrics(config)
			case <-monitorStopCh:
				return
			}
		}
	}()
}

// StopGlobalMonitor 停止全局协程池监控
func StopGlobalMonitor() {
	monitorMu.Lock()
	defer monitorMu.Unlock()

	if !monitorRunning {
		return
	}

	if monitorTicker != nil {
		monitorTicker.Stop()
	}
	if monitorStopCh != nil {
		close(monitorStopCh)
	}

	monitorRunning = false
	log.Printf("[ants] 监控已停止")
}

// printMonitorMetrics 打印监控指标
func printMonitorMetrics(config MonitorConfig) {
	if antsPool == nil {
		return
	}

	metrics := antsPool.Metrics()

	// 如果有自定义打印函数，使用自定义函数
	if config.CustomPrinter != nil {
		config.CustomPrinter(metrics)
		return
	}

	// 打印基础指标
	if config.PrintBasicMetrics {
		log.Printf("[ants][%s] 监控 - 运行协程数: %d, 等待任务数: %d, 总任务数: %d, 成功: %d, 失败: %d, 池容量: %d",
			metrics.Name, metrics.RunningGoroutines, metrics.WaitingTasks,
			metrics.TotalTasks, metrics.SuccessTasks, metrics.FailedTasks, metrics.PoolCapacity)

		log.Printf("[ants][%s] 监控 - 平均执行时间: %v, 最大执行时间: %v, 最小执行时间: %v",
			metrics.Name, metrics.TaskExecutionTime, metrics.MaxTaskExecutionTime, metrics.MinTaskExecutionTime)
	}

	// 打印健康分析信息
	if config.PrintHealthInfo {
		log.Printf("[ants][%s] 健康分析 - 被追踪任务数: %d, 慢任务数: %d, 异常任务数: %d",
			metrics.Name, metrics.TrackedTasks, len(metrics.SlowTasks), len(metrics.FailedTasksList))

		// 打印慢任务详情
		if len(metrics.SlowTasks) > 0 {
			log.Printf("[ants][%s] ⚠ 发现 %d 个慢任务:", metrics.Name, len(metrics.SlowTasks))
			maxPrint := config.MaxSlowTasksPrint
			if maxPrint > len(metrics.SlowTasks) {
				maxPrint = len(metrics.SlowTasks)
			}
			for i := 0; i < maxPrint; i++ {
				task := metrics.SlowTasks[i]
				log.Printf("  - 任务ID: %s, 执行时间: %v, 记录时间: %s",
					task.TaskID, task.ExecutionTime, task.Timestamp.Format("15:04:05"))
			}
			if len(metrics.SlowTasks) > maxPrint {
				log.Printf("  ... 还有 %d 个慢任务", len(metrics.SlowTasks)-maxPrint)
			}
		}

		// 打印异常任务详情
		if len(metrics.FailedTasksList) > 0 {
			log.Printf("[ants][%s] ✗ 发现 %d 个异常任务:", metrics.Name, len(metrics.FailedTasksList))
			maxPrint := config.MaxFailedTasksPrint
			if maxPrint > len(metrics.FailedTasksList) {
				maxPrint = len(metrics.FailedTasksList)
			}
			for i := 0; i < maxPrint; i++ {
				task := metrics.FailedTasksList[i]
				log.Printf("  - 任务ID: %s, 错误: %s, 记录时间: %s",
					task.TaskID, task.Error, task.Timestamp.Format("15:04:05"))
			}
			if len(metrics.FailedTasksList) > maxPrint {
				log.Printf("  ... 还有 %d 个异常任务", len(metrics.FailedTasksList)-maxPrint)
			}
		}
	}
}

// SetMonitorConfig 动态设置监控配置（会重启监控）
func SetMonitorConfig(config MonitorConfig) {
	if config.Enabled {
		StartGlobalMonitor(config)
	} else {
		StopGlobalMonitor()
	}
}

// IsMonitorRunning 检查监控是否正在运行
func IsMonitorRunning() bool {
	monitorMu.Lock()
	defer monitorMu.Unlock()
	return monitorRunning
}
