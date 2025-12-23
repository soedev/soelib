package ants

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// TestInitCoroutinePoolWithFullConfig 测试使用完整配置初始化全局池
func TestInitCoroutinePoolWithFullConfig(t *testing.T) {
	// 注意：由于使用了 sync.Once，这个测试需要在独立进程中运行
	// 这里我们测试配置结构的创建
	config := GlobalPoolConfig{
		Name:     "test-global-pool",
		Size:     10,
		PreAlloc: true,
		Health: HealthConfig{
			Enabled:              true,
			SlowTaskThreshold:    time.Millisecond * 500,
			MaxSlowTaskRecords:   50,
			MaxFailedTaskRecords: 50,
		},
	}

	// 验证配置结构
	if config.Name != "test-global-pool" {
		t.Errorf("期望名称为 'test-global-pool'，实际为 '%s'", config.Name)
	}
	if config.Size != 10 {
		t.Errorf("期望大小为 10，实际为 %d", config.Size)
	}
	if !config.Health.Enabled {
		t.Error("期望健康分析启用")
	}
	if config.Health.SlowTaskThreshold != time.Millisecond*500 {
		t.Errorf("期望慢任务阈值为 500ms，实际为 %v", config.Health.SlowTaskThreshold)
	}

	t.Logf("全局池配置结构验证通过")
}

// TestGlobalPoolMethods 测试全局池方法
func TestGlobalPoolMethods(t *testing.T) {
	// 初始化全局池（如果还未初始化）
	InitCoroutinePool()

	// 等待初始化完成
	time.Sleep(time.Millisecond * 100)

	// 测试获取全局池
	pool := GetPool()
	if pool == nil {
		t.Fatal("全局池不应该为 nil")
	}

	// 测试设置健康配置
	healthConfig := HealthConfig{
		Enabled:              true,
		SlowTaskThreshold:    time.Millisecond * 300,
		MaxSlowTaskRecords:   20,
		MaxFailedTaskRecords: 20,
	}
	SetGlobalHealthConfig(healthConfig)

	// 验证配置已设置
	retrievedConfig := GetGlobalHealthConfig()
	if !retrievedConfig.Enabled {
		t.Error("健康分析应该被启用")
	}
	if retrievedConfig.SlowTaskThreshold != time.Millisecond*300 {
		t.Errorf("期望慢任务阈值为 300ms，实际为 %v", retrievedConfig.SlowTaskThreshold)
	}

	// 测试提交普通任务
	var normalTaskCount int32
	for i := 0; i < 5; i++ {
		SubmitTask(func() {
			atomic.AddInt32(&normalTaskCount, 1)
			time.Sleep(time.Millisecond * 50)
		})
	}

	// 测试提交带ID的任务
	var trackedTaskCount int32
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("global-task-%d", i)
		SubmitTaskWithID(taskID, func() {
			atomic.AddInt32(&trackedTaskCount, 1)
			time.Sleep(time.Millisecond * 100)
		})
	}

	// 提交慢任务
	for i := 0; i < 2; i++ {
		taskID := fmt.Sprintf("global-slow-task-%d", i)
		SubmitTaskWithID(taskID, func() {
			time.Sleep(time.Millisecond * 400) // 超过阈值
		})
	}

	// 提交异常任务
	taskID := "global-failed-task"
	SubmitTaskWithID(taskID, func() {
		panic("模拟全局任务异常")
	})

	// 等待任务完成
	time.Sleep(time.Second * 2)

	// 获取全局指标
	metrics := GetGlobalMetrics()

	t.Logf("全局池指标:")
	t.Logf("  总任务数: %d", metrics.TotalTasks)
	t.Logf("  被追踪任务数: %d", metrics.TrackedTasks)
	t.Logf("  慢任务数: %d", len(metrics.SlowTasks))
	t.Logf("  异常任务数: %d", len(metrics.FailedTasksList))

	// 验证任务执行
	if normalTaskCount != 5 {
		t.Errorf("期望执行 5 个普通任务，实际执行 %d 个", normalTaskCount)
	}
	if trackedTaskCount != 3 {
		t.Errorf("期望执行 3 个追踪任务，实际执行 %d 个", trackedTaskCount)
	}

	// 验证健康分析
	if metrics.TrackedTasks < 6 { // 3个普通追踪任务 + 2个慢任务 + 1个异常任务
		t.Errorf("期望至少追踪 6 个任务，实际追踪 %d 个", metrics.TrackedTasks)
	}

	if len(metrics.SlowTasks) != 2 {
		t.Errorf("期望记录 2 个慢任务，实际记录 %d 个", len(metrics.SlowTasks))
	}

	if len(metrics.FailedTasksList) != 1 {
		t.Errorf("期望记录 1 个异常任务，实际记录 %d 个", len(metrics.FailedTasksList))
	}

	// 打印慢任务详情
	for _, task := range metrics.SlowTasks {
		t.Logf("  慢任务: %s, 执行时间: %v", task.TaskID, task.ExecutionTime)
	}

	// 打印异常任务详情
	for _, task := range metrics.FailedTasksList {
		t.Logf("  异常任务: %s, 错误: %s", task.TaskID, task.Error)
	}
}

// TestAutoMonitor 测试自动监控功能
func TestAutoMonitor(t *testing.T) {
	// 创建配置（启用自动监控）
	config := GlobalPoolConfig{
		Name:     "monitor-test-pool",
		Size:     10,
		PreAlloc: true,
		Health: HealthConfig{
			Enabled:              true,
			SlowTaskThreshold:    time.Millisecond * 200,
			MaxSlowTaskRecords:   50,
			MaxFailedTaskRecords: 50,
		},
		Monitor: MonitorConfig{
			Enabled:             true,
			Interval:            time.Second * 2, // 2秒打印一次（测试用）
			PrintBasicMetrics:   true,
			PrintHealthInfo:     true,
			MaxSlowTasksPrint:   3,
			MaxFailedTasksPrint: 3,
		},
	}

	// 注意：由于全局池使用 sync.Once，这里只能测试配置
	t.Logf("监控配置创建成功")
	t.Logf("  启用: %t", config.Monitor.Enabled)
	t.Logf("  间隔: %v", config.Monitor.Interval)
	t.Logf("  打印基础指标: %t", config.Monitor.PrintBasicMetrics)
	t.Logf("  打印健康信息: %t", config.Monitor.PrintHealthInfo)
}

// TestCustomMonitorPrinter 测试自定义打印函数
func TestCustomMonitorPrinter(t *testing.T) {
	customPrinted := false

	config := MonitorConfig{
		Enabled:  true,
		Interval: time.Second,
		CustomPrinter: func(metrics Metrics) {
			customPrinted = true
			t.Logf("自定义打印 - 总任务数: %d, 成功: %d, 失败: %d",
				metrics.TotalTasks, metrics.SuccessTasks, metrics.FailedTasks)
		},
	}

	t.Logf("自定义打印配置创建成功")
	if config.CustomPrinter != nil {
		t.Logf("  自定义打印函数已设置")

		// 模拟调用
		config.CustomPrinter(Metrics{
			Name:         "test",
			TotalTasks:   100,
			SuccessTasks: 95,
			FailedTasks:  5,
		})

		if customPrinted {
			t.Logf("  自定义打印函数执行成功")
		}
	}
}
