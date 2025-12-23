package ants

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPoolCreation 测试线程池创建
func TestPoolCreation(t *testing.T) {
	// 测试默认线程池
	pool, err := New("test-pool")
	if err != nil {
		t.Fatalf("创建默认线程池失败: %v", err)
	}
	defer pool.Release()

	// 测试自定义大小线程池
	pool, err = New("test-pool", 10)
	if err != nil {
		t.Fatalf("创建自定义大小线程池失败: %v", err)
	}
	defer pool.Release()
}

// TestTaskSubmission 测试任务提交
func TestTaskSubmission(t *testing.T) {
	pool, err := New("test-pool", 5)
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	var count int
	var mu sync.Mutex

	// 提交10个任务
	for i := 0; i < 10; i++ {
		if err := pool.Submit(func() {
			mu.Lock()
			count++
			mu.Unlock()
			time.Sleep(time.Millisecond * 10)
		}); err != nil {
			t.Fatalf("提交任务失败: %v", err)
		}
	}

	// 等待所有任务完成
	time.Sleep(time.Millisecond * 200)

	mu.Lock()
	defer mu.Unlock()
	if count != 10 {
		t.Errorf("期望完成10个任务，实际完成%d个", count)
	}
}

// TestSubmitWithContext 测试上下文感知任务提交
func TestSubmitWithContext(t *testing.T) {
	pool, err := New("test-pool", 5)
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 创建一个短超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	var completed bool

	if err := pool.SubmitWithContext(ctx, func(ctx context.Context) {
		// 模拟定期检查上下文的长时间运行任务
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				// 上下文已取消，提前返回
				return
			default:
				// 执行一些工作
				time.Sleep(time.Millisecond * 10)
			}
		}
		completed = true
	}); err != nil {
		t.Fatalf("提交上下文感知任务失败: %v", err)
	}

	// 等待并检查任务是否被取消
	time.Sleep(time.Millisecond * 200)
	if completed {
		t.Error("任务应该因上下文超时而被取消")
	}
}

// TestSubmitWithTimeout 测试带超时的任务提交
func TestSubmitWithTimeout(t *testing.T) {
	pool, err := New("test-pool", 5)
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	var completed bool

	if err := pool.SubmitWithTimeout(time.Millisecond*50, func() {
		// 模拟长时间运行任务
		time.Sleep(time.Millisecond * 100)
		completed = true
	}); err != nil {
		t.Fatalf("提交超时任务失败: %v", err)
	}

	// 等待并检查任务是否被取消
	time.Sleep(time.Millisecond * 200)
	if completed {
		t.Error("任务应该因超时而被取消")
	}
}

// TestPoolWithFunc 测试 PoolWithFunc 功能
func TestPoolWithFunc(t *testing.T) {
	var sum int
	var mu sync.Mutex

	pool, err := NewPoolWithFunc("test-pool-with-func", func(arg interface{}) {
		mu.Lock()
		sum += arg.(int)
		mu.Unlock()
		time.Sleep(time.Millisecond * 10)
	}, 5)
	if err != nil {
		t.Fatalf("创建带函数的线程池失败: %v", err)
	}
	defer pool.Release()

	// 调用10个任务
	for i := 1; i <= 10; i++ {
		if err := pool.Invoke(i); err != nil {
			t.Fatalf("调用任务失败: %v", err)
		}
	}

	// 等待所有任务完成
	time.Sleep(time.Millisecond * 200)

	expectedSum := 55 // 1+2+...+10
	mu.Lock()
	defer mu.Unlock()
	if sum != expectedSum {
		t.Errorf("期望和为%d，实际为%d", expectedSum, sum)
	}
}

// TestMetrics 测试指标收集
func TestMetrics(t *testing.T) {
	pool, err := New("test-pool", 5)
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 提交一些任务
	for i := 0; i < 3; i++ {
		if err := pool.Submit(func() {
			time.Sleep(time.Millisecond * 50)
		}); err != nil {
			t.Fatalf("提交任务失败: %v", err)
		}
	}

	// 获取指标
	metrics := pool.Metrics()
	if metrics.Name != "test-pool" {
		t.Errorf("期望线程池名称为'test-pool'，实际为'%s'", metrics.Name)
	}
	if metrics.TotalTasks < 3 {
		t.Errorf("期望至少有3个总任务，实际有%d个", metrics.TotalTasks)
	}
}

// TestPanicRecovery 测试panic恢复
func TestPanicRecovery(t *testing.T) {
	pool, err := New("test-pool", 5)
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 提交一个会panic的任务
	if err := pool.Submit(func() {
		panic("测试panic")
	}); err != nil {
		t.Fatalf("提交panic任务失败: %v", err)
	}

	// 等待panic被处理
	time.Sleep(time.Millisecond * 100)

	// 提交另一个任务以确保线程池仍在工作
	var completed bool
	if err := pool.Submit(func() {
		completed = true
	}); err != nil {
		t.Fatalf("panic后提交任务失败: %v", err)
	}

	time.Sleep(time.Millisecond * 100)
	if !completed {
		t.Error("panic后线程池应该仍能正常工作")
	}
}

// TestErrorCallback 测试错误回调函数
func TestErrorCallback(t *testing.T) {
	pool, err := New("test-pool", 1) // 使用小容量池，方便测试
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 注册错误回调函数
	var callbackErrCode int
	callbackCh := make(chan struct{}, 1)
	pool.RegisterErrorCallback(func(err *PoolError) {
		callbackErrCode = err.Code
		callbackCh <- struct{}{}
	})

	// 测试1: 提交一个会panic的任务，验证错误回调函数是否被调用
	if err := pool.Submit(func() {
		panic("测试panic")
	}); err != nil {
		t.Fatalf("提交panic任务失败: %v", err)
	}

	// 等待panic被处理和回调函数被调用
	select {
	case <-callbackCh:
		// 回调函数被调用，继续测试
	case <-time.After(time.Second * 2):
		// 超时，回调函数没有被调用
		t.Error("错误回调函数应该被调用")
		return
	}

	if callbackErrCode != ErrCodeTaskPanic {
		t.Errorf("期望错误码为%d，实际为%d", ErrCodeTaskPanic, callbackErrCode)
	}

	// 测试2: 测试错误码是否正确
	// 提交一个任务，验证错误码是否正确
	if err := pool.Submit(func() {
		time.Sleep(time.Millisecond * 50)
	}); err != nil {
		// 这个任务应该成功提交，所以这里不应该有错误
		t.Fatalf("提交正常任务失败: %v", err)
	}

	// 等待任务完成
	time.Sleep(time.Millisecond * 100)
}

// TestPoolError 测试PoolError结构体
func TestPoolError(t *testing.T) {
	// 创建一个PoolError实例
	underlyingErr := fmt.Errorf("底层错误")
	poolErr := &PoolError{
		Code:    ErrCodeSubmitTask,
		Message: "测试错误",
		Err:     underlyingErr,
	}

	// 测试Error方法
	expectedError := "测试错误: 底层错误"
	if poolErr.Error() != expectedError {
		t.Errorf("期望错误消息为%s，实际为%s", expectedError, poolErr.Error())
	}

	// 测试Unwrap方法
	if !errors.Is(underlyingErr, poolErr.Unwrap()) {
		t.Error("Unwrap方法应该返回底层错误")
	}

	// 测试没有底层错误的情况
	poolErr2 := &PoolError{
		Code:    ErrCodeCreatePool,
		Message: "测试错误2",
		Err:     nil,
	}

	expectedError2 := "测试错误2"
	if poolErr2.Error() != expectedError2 {
		t.Errorf("期望错误消息为%s，实际为%s", expectedError2, poolErr2.Error())
	}

	if poolErr2.Unwrap() != nil {
		t.Error("Unwrap方法应该返回nil当没有底层错误时")
	}
}

// BenchmarkMetricsPerformance 测试健康指标性能
func BenchmarkMetricsPerformance(b *testing.B) {
	// 创建一个大型线程池
	pool, err := New("benchmark-pool", 100)
	if err != nil {
		b.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 用于等待所有任务完成的wg
	var wg sync.WaitGroup

	// 重置计时器
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 提交大量任务
		for j := 0; j < 1000; j++ {
			wg.Add(1)
			if err := pool.Submit(func() {
				defer wg.Done()
				// 模拟一个简单的任务
				_ = 1 + 1
			}); err != nil {
				b.Fatalf("提交任务失败: %v", err)
			}
		}

		// 获取健康指标
		metrics := pool.Metrics()

		// 验证指标是否合理
		if metrics.Name != "benchmark-pool" {
			b.Errorf("期望线程池名称为'benchmark-pool'，实际为'%s'", metrics.Name)
		}
		if metrics.TotalTasks == 0 {
			b.Error("期望总任务数大于0")
		}
	}

	// 等待所有任务完成
	wg.Wait()
}

// TestMetricsPerformance 测试健康指标在高并发场景下的表现
func TestMetricsPerformance(t *testing.T) {
	// 创建线程池，使用默认配置即可
	pool, err := New("performance-test-pool", 50)
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 用于等待所有任务完成的wg
	var wg sync.WaitGroup

	// 提交大量任务
	totalTasks := 10000
	wg.Add(totalTasks)

	startTime := time.Now()

	// 提交任务（不使用过多协程）
	for i := 0; i < totalTasks; i++ {
		if err := pool.Submit(func() {
			defer wg.Done()
			// 模拟一个简单的计算任务
			result := 0
			for j := 0; j < 1000; j++ {
				result += j
			}
			_ = result
		}); err != nil {
			t.Errorf("提交任务失败: %v", err)
		}
	}

	// 定期获取并打印健康指标
	metricsTicker := time.NewTicker(time.Millisecond * 100)
	defer metricsTicker.Stop()

	metricsCh := make(chan Metrics, 10)
	stopCh := make(chan struct{})

	// 监控协程，定期获取指标
	go func() {
		for {
			select {
			case <-metricsTicker.C:
				metrics := pool.Metrics()
				select {
				case metricsCh <- metrics:
				default:
					// 通道已满，跳过这个指标
				}
			case <-stopCh:
				// 收到停止信号，退出
				return
			}
		}
	}()

	// 收集指标
	var allMetrics []Metrics
	metricsReceived := 0
	metricsDoneCh := make(chan struct{})

	// 指标收集协程
	go func() {
		for metrics := range metricsCh {
			allMetrics = append(allMetrics, metrics)
			metricsReceived++
		}
		close(metricsDoneCh)
	}()

	// 等待所有任务完成或超时
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		// 所有任务完成
	case <-time.After(time.Second * 10):
		t.Error("任务执行超时")
	}

	// 停止监控
	close(stopCh)
	metricsTicker.Stop()
	close(metricsCh)

	// 等待指标收集完成
	<-metricsDoneCh

	totalTime := time.Since(startTime)

	// 打印性能报告
	t.Logf("测试完成:")
	t.Logf("总任务数: %d", totalTasks)
	t.Logf("总执行时间: %v", totalTime)
	t.Logf("任务吞吐量: %.2f 任务/秒", float64(totalTasks)/totalTime.Seconds())
	t.Logf("收集到的指标数量: %d", metricsReceived)

	// 验证最终指标
	finalMetrics := pool.Metrics()
	t.Logf("最终指标:")
	t.Logf("  运行中的协程数: %d", finalMetrics.RunningGoroutines)
	t.Logf("  等待中的任务数: %d", finalMetrics.WaitingTasks)
	t.Logf("  总任务数: %d", finalMetrics.TotalTasks)
	t.Logf("  成功任务数: %d", finalMetrics.SuccessTasks)
	t.Logf("  失败任务数: %d", finalMetrics.FailedTasks)
	t.Logf("  平均任务执行时间: %v", finalMetrics.TaskExecutionTime)
	t.Logf("  最大任务执行时间: %v", finalMetrics.MaxTaskExecutionTime)
	t.Logf("  最小任务执行时间: %v", finalMetrics.MinTaskExecutionTime)
	t.Logf("  协程创建总数: %d", finalMetrics.GoroutinesCreated)
	t.Logf("  协程销毁总数: %d", finalMetrics.GoroutinesDestroyed)

	// 验证指标是否合理
	expectedMinTasks := int64(totalTasks * 95 / 100) // 使用整数计算避免浮点数问题
	if finalMetrics.TotalTasks < expectedMinTasks {
		t.Errorf("期望总任务数至少为%d，实际为%d", expectedMinTasks, finalMetrics.TotalTasks)
	}
	if finalMetrics.SuccessTasks < finalMetrics.TotalTasks-100 {
		t.Errorf("期望失败任务数不超过100，实际为%d", finalMetrics.TotalTasks-finalMetrics.SuccessTasks)
	}
}

// TestHealthAnalysis 测试健康分析功能
func TestHealthAnalysis(t *testing.T) {
	// 创建线程池
	pool, err := New("health-test-pool", 5)
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 配置健康分析
	healthConfig := HealthConfig{
		Enabled:              true,
		SlowTaskThreshold:    time.Millisecond * 200, // 超过200ms视为慢任务
		MaxSlowTaskRecords:   10,
		MaxFailedTaskRecords: 10,
	}
	pool.SetHealthConfig(healthConfig)

	// 验证配置
	config := pool.GetHealthConfig()
	if !config.Enabled {
		t.Error("健康分析应该被启用")
	}
	if config.SlowTaskThreshold != time.Millisecond*200 {
		t.Errorf("慢任务阈值应该为200ms，实际为%v", config.SlowTaskThreshold)
	}

	// 提交正常任务（不追踪）
	var normalTaskCount int32
	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			atomic.AddInt32(&normalTaskCount, 1)
			time.Sleep(time.Millisecond * 50)
		})
	}

	// 提交快速任务（追踪但不是慢任务）
	var fastTaskCount int32
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("fast-task-%d", i)
		pool.SubmitWithTaskID(taskID, func() {
			atomic.AddInt32(&fastTaskCount, 1)
			time.Sleep(time.Millisecond * 50)
		})
	}

	// 提交慢任务
	var slowTaskCount int32
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("slow-task-%d", i)
		pool.SubmitWithTaskID(taskID, func() {
			atomic.AddInt32(&slowTaskCount, 1)
			time.Sleep(time.Millisecond * 300) // 超过阈值
		})
	}

	// 提交异常任务
	var failedTaskCount int32
	for i := 0; i < 2; i++ {
		taskID := fmt.Sprintf("failed-task-%d", i)
		pool.SubmitWithTaskID(taskID, func() {
			atomic.AddInt32(&failedTaskCount, 1)
			panic(fmt.Sprintf("模拟任务异常: %s", taskID))
		})
	}

	// 等待所有任务完成
	time.Sleep(time.Second * 2)

	// 获取健康指标
	metrics := pool.Metrics()

	// 验证基本指标
	t.Logf("总任务数: %d", metrics.TotalTasks)
	t.Logf("被追踪任务数: %d", metrics.TrackedTasks)
	t.Logf("失败任务数: %d", metrics.FailedTasks)

	// 验证追踪任务数（应该是快速任务+慢任务+异常任务）
	expectedTrackedTasks := int64(3 + 3 + 2)
	if metrics.TrackedTasks != expectedTrackedTasks {
		t.Errorf("期望追踪任务数为%d，实际为%d", expectedTrackedTasks, metrics.TrackedTasks)
	}

	// 验证慢任务
	t.Logf("慢任务数: %d", len(metrics.SlowTasks))
	if len(metrics.SlowTasks) != 3 {
		t.Errorf("期望慢任务数为3，实际为%d", len(metrics.SlowTasks))
	}

	// 打印慢任务详情
	for _, task := range metrics.SlowTasks {
		t.Logf("  慢任务: %s, 执行时间: %v, 记录时间: %s",
			task.TaskID, task.ExecutionTime, task.Timestamp.Format("15:04:05"))
		if task.ExecutionTime <= healthConfig.SlowTaskThreshold {
			t.Errorf("任务%s执行时间%v应该超过阈值%v", task.TaskID, task.ExecutionTime, healthConfig.SlowTaskThreshold)
		}
	}

	// 验证异常任务
	t.Logf("异常任务数: %d", len(metrics.FailedTasksList))
	if len(metrics.FailedTasksList) != 2 {
		t.Errorf("期望异常任务数为2，实际为%d", len(metrics.FailedTasksList))
	}

	// 打印异常任务详情
	for _, task := range metrics.FailedTasksList {
		t.Logf("  异常任务: %s, 错误: %s, 记录时间: %s",
			task.TaskID, task.Error, task.Timestamp.Format("15:04:05"))
	}

	// 验证失败任务数
	if metrics.FailedTasks != 2 {
		t.Errorf("期望失败任务数为2，实际为%d", metrics.FailedTasks)
	}
}

// TestHealthAnalysisWithPoolWithFunc 测试 PoolWithFunc 的健康分析功能
func TestHealthAnalysisWithPoolWithFunc(t *testing.T) {
	// 创建带函数的线程池
	var processedCount int32
	wfPool, err := NewPoolWithFunc("health-test-wf-pool", func(arg interface{}) {
		atomic.AddInt32(&processedCount, 1)

		taskInfo := arg.(map[string]interface{})
		duration := taskInfo["duration"].(time.Duration)
		shouldPanic := taskInfo["panic"].(bool)

		time.Sleep(duration)

		if shouldPanic {
			panic("模拟任务异常")
		}
	}, 5)
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer wfPool.Release()

	// 配置健康分析
	healthConfig := HealthConfig{
		Enabled:              true,
		SlowTaskThreshold:    time.Millisecond * 150,
		MaxSlowTaskRecords:   10,
		MaxFailedTaskRecords: 10,
	}
	wfPool.SetHealthConfig(healthConfig)

	// 提交不追踪的任务
	wfPool.Invoke(map[string]interface{}{
		"duration": time.Millisecond * 50,
		"panic":    false,
	})

	// 提交快速任务（追踪）
	wfPool.InvokeWithTaskID("fast-task", map[string]interface{}{
		"duration": time.Millisecond * 50,
		"panic":    false,
	})

	// 提交慢任务（追踪）
	wfPool.InvokeWithTaskID("slow-task", map[string]interface{}{
		"duration": time.Millisecond * 200,
		"panic":    false,
	})

	// 提交异常任务（追踪）
	wfPool.InvokeWithTaskID("failed-task", map[string]interface{}{
		"duration": time.Millisecond * 50,
		"panic":    true,
	})

	// 等待任务完成
	time.Sleep(time.Second * 1)

	// 获取健康指标
	metrics := wfPool.Metrics()

	t.Logf("总任务数: %d", metrics.TotalTasks)
	t.Logf("被追踪任务数: %d", metrics.TrackedTasks)
	t.Logf("慢任务数: %d", len(metrics.SlowTasks))
	t.Logf("异常任务数: %d", len(metrics.FailedTasksList))

	// 验证追踪任务数
	if metrics.TrackedTasks != 3 {
		t.Errorf("期望追踪任务数为3，实际为%d", metrics.TrackedTasks)
	}

	// 验证慢任务
	if len(metrics.SlowTasks) != 1 {
		t.Errorf("期望慢任务数为1，实际为%d", len(metrics.SlowTasks))
	}

	// 验证异常任务
	if len(metrics.FailedTasksList) != 1 {
		t.Errorf("期望异常任务数为1，实际为%d", len(metrics.FailedTasksList))
	}
}

// TestHealthAnalysisDisabled 测试禁用健康分析的情况
func TestHealthAnalysisDisabled(t *testing.T) {
	// 创建线程池
	pool, err := New("disabled-health-pool", 5)
	if err != nil {
		t.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 默认情况下健康分析应该是禁用的
	config := pool.GetHealthConfig()
	if config.Enabled {
		t.Error("健康分析默认应该被禁用")
	}

	// 提交带ID的任务
	for i := 0; i < 5; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		pool.SubmitWithTaskID(taskID, func() {
			time.Sleep(time.Millisecond * 100)
		})
	}

	// 等待任务完成
	time.Sleep(time.Second * 1)

	// 获取指标
	metrics := pool.Metrics()

	// 当健康分析禁用时，不应该记录慢任务和异常任务
	if len(metrics.SlowTasks) != 0 {
		t.Errorf("健康分析禁用时不应该记录慢任务，实际记录了%d个", len(metrics.SlowTasks))
	}
	if len(metrics.FailedTasksList) != 0 {
		t.Errorf("健康分析禁用时不应该记录异常任务，实际记录了%d个", len(metrics.FailedTasksList))
	}
}
