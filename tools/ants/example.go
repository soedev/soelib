package ants

import (
	"context"
	"fmt"
	"log"
	"time"

	antslib "github.com/panjf2000/ants/v2"
)

// Example 演示 ants 包的使用
func Example() {
	// 1. 创建一个新的线程池，使用默认配置
	pool, err := New("example-pool")
	if err != nil {
		log.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 1.1 使用自定义配置创建线程池
	// 演示如何使用自定义配置，包括预分配内存、自定义队列大小等
	customPool, err := NewWithOptions("custom-pool", 10, func(opts *antslib.Options) {
		opts.PreAlloc = true                  // 启用预分配内存
		opts.MaxBlockingTasks = 1000          // 自定义队列大小
		opts.ExpiryDuration = 5 * time.Minute // 自定义过期时间
	})
	if err != nil {
		log.Fatalf("创建自定义配置线程池失败: %v", err)
	}
	defer customPool.Release()
	log.Printf("创建自定义配置线程池成功，预分配: true, 队列大小: 1000, 过期时间: 5分钟")

	// 2. 基本任务提交
	for i := 0; i < 10; i++ {
		// 注意：使用局部变量副本避免闭包陷阱
		taskID := i
		if err := pool.Submit(func() {
			log.Printf("处理任务 %d", taskID)
			time.Sleep(time.Millisecond * 100)
		}); err != nil {
			log.Printf("提交任务 %d 失败: %v", taskID, err)
		}
	}

	// 3. 带有上下文的任务提交
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := pool.SubmitWithContext(ctx, func(ctx context.Context) {
		// 模拟长时间运行的任务
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				log.Println("任务因上下文取消而终止")
				return
			default:
				log.Println("仍在工作...")
				time.Sleep(time.Millisecond * 500)
			}
		}
	}); err != nil {
		log.Printf("提交上下文感知任务失败: %v", err)
	}

	// 4. 带有超时的任务提交
	if err := pool.SubmitWithTimeout(time.Second*2, func() {
		// 模拟可能超时的任务
		time.Sleep(time.Millisecond * 3000) // 这会导致超时
		log.Println("任务超时后完成")
	}); err != nil {
		log.Printf("提交超时任务失败: %v", err)
	}

	// 5. 注意：由于 Go 语言不支持在方法上使用类型参数，
	// 我们暂时移除了 SubmitWithResult 相关的泛型实现
	// 后续可以考虑使用其他方式实现，例如使用 interface{} 或单独的函数
	// 以下是移除的示例代码：
	/*
		// 提交带有结果的任务
		// 演示如何使用 SubmitWithResult 方法提交带有结果的任务
		future, err := pool.SubmitWithResult(func() (int, error) {
			time.Sleep(time.Millisecond * 500)
			return 42, nil
		})
		if err != nil {
			log.Printf("提交带结果任务失败: %v", err)
		} else {
			// 获取任务执行结果
			result, err := future.Get()
			if err != nil {
				log.Printf("获取任务结果失败: %v", err)
			} else {
				log.Printf("获取到任务结果: %d", result)
			}
		}

		// 提交带有上下文和结果的任务
		futureWithContext, err := pool.SubmitWithResultWithContext(ctx, func(ctx context.Context) (string, error) {
			for i := 0; i < 5; i++ {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				default:
					time.Sleep(time.Millisecond * 200)
				}
			}
			return "任务完成", nil
		})
		if err != nil {
			log.Printf("提交带上下文和结果任务失败: %v", err)
		} else {
			// 带超时获取任务执行结果
			result, err := futureWithContext.GetWithTimeout(1 * time.Second)
			if err != nil {
				log.Printf("带超时获取任务结果失败: %v", err)
			} else {
				log.Printf("带超时获取到任务结果: %s", result)
			}
		}
	*/

	// 6. 使用PoolWithFunc
	wfPool, err := NewPoolWithFunc("worker-pool", func(arg interface{}) {
		log.Printf("工作协程处理: %v", arg)
		time.Sleep(time.Millisecond * 50)
	})
	if err != nil {
		log.Fatalf("创建带函数的线程池失败: %v", err)
	}
	defer wfPool.Release()

	// 提交任务到PoolWithFunc
	for i := 0; i < 5; i++ {
		if err := wfPool.Invoke(i); err != nil {
			log.Printf("调用工作协程任务 %d 失败: %v", i, err)
		}
	}

	// 7. 监控池状态
	time.Sleep(time.Second)
	metrics := pool.Metrics()
	log.Printf("线程池指标: %+v", metrics)

	wfMetrics := wfPool.Metrics()
	log.Printf("工作协程池指标: %+v", wfMetrics)

	// 7.1 展示更详细的监控指标
	customMetrics := customPool.Metrics()
	log.Printf("自定义配置线程池详细指标:")
	log.Printf("  名称: %s", customMetrics.Name)
	log.Printf("  运行协程数: %d", customMetrics.RunningGoroutines)
	log.Printf("  等待任务数: %d", customMetrics.WaitingTasks)
	log.Printf("  总任务数: %d", customMetrics.TotalTasks)
	log.Printf("  成功任务数: %d", customMetrics.SuccessTasks)
	log.Printf("  失败任务数: %d", customMetrics.FailedTasks)
	log.Printf("  池容量: %d", customMetrics.PoolCapacity)
	log.Printf("  平均任务执行时间: %v", customMetrics.TaskExecutionTime)
	log.Printf("  最大任务执行时间: %v", customMetrics.MaxTaskExecutionTime)
	log.Printf("  最小任务执行时间: %v", customMetrics.MinTaskExecutionTime)
	log.Printf("  队列长度变化: %d", customMetrics.QueueLengthChange)
}

// HealthAnalysisExample 演示健康分析功能的使用
func HealthAnalysisExample() {
	// 1. 创建一个启用健康分析的线程池
	pool, err := New("health-pool", 10)
	if err != nil {
		log.Fatalf("创建线程池失败: %v", err)
	}
	defer pool.Release()

	// 2. 配置健康分析
	healthConfig := HealthConfig{
		Enabled:              true,                   // 启用健康分析
		SlowTaskThreshold:    time.Millisecond * 500, // 超过500ms视为慢任务
		MaxSlowTaskRecords:   50,                     // 最多记录50个慢任务
		MaxFailedTaskRecords: 50,                     // 最多记录50个异常任务
	}
	pool.SetHealthConfig(healthConfig)
	log.Printf("健康分析已启用，慢任务阈值: %v", healthConfig.SlowTaskThreshold)

	// 3. 提交正常任务（不会被追踪）
	for i := 0; i < 5; i++ {
		taskID := i
		pool.Submit(func() {
			log.Printf("正常任务 %d 执行中（不会被追踪）", taskID)
			time.Sleep(time.Millisecond * 100)
		})
	}

	// 4. 提交带ID的快速任务（会被追踪，但不是慢任务）
	for i := 0; i < 5; i++ {
		taskID := fmt.Sprintf("fast-task-%d", i)
		pool.SubmitWithTaskID(taskID, func() {
			log.Printf("快速任务 %s 执行中", taskID)
			time.Sleep(time.Millisecond * 100)
		})
	}

	// 5. 提交带ID的慢任务（会被记录为慢任务）
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("slow-task-%d", i)
		pool.SubmitWithTaskID(taskID, func() {
			log.Printf("慢任务 %s 执行中", taskID)
			time.Sleep(time.Millisecond * 800) // 超过阈值
		})
	}

	// 6. 提交带ID的异常任务（会被记录为异常任务）
	for i := 0; i < 2; i++ {
		taskID := fmt.Sprintf("failed-task-%d", i)
		pool.SubmitWithTaskID(taskID, func() {
			log.Printf("异常任务 %s 执行中", taskID)
			panic(fmt.Sprintf("模拟任务 %s 发生异常", taskID))
		})
	}

	// 7. 等待任务执行完成
	time.Sleep(time.Second * 3)

	// 8. 获取健康指标并打印分析结果
	metrics := pool.Metrics()
	log.Printf("\n========== 健康分析报告 ==========")
	log.Printf("线程池名称: %s", metrics.Name)
	log.Printf("总任务数: %d", metrics.TotalTasks)
	log.Printf("被追踪任务数: %d", metrics.TrackedTasks)
	log.Printf("成功任务数: %d", metrics.SuccessTasks)
	log.Printf("失败任务数: %d", metrics.FailedTasks)
	log.Printf("平均执行时间: %v", metrics.TaskExecutionTime)
	log.Printf("最大执行时间: %v", metrics.MaxTaskExecutionTime)
	log.Printf("最小执行时间: %v", metrics.MinTaskExecutionTime)

	// 9. 打印慢任务分析
	log.Printf("\n--- 慢任务分析 (阈值: %v) ---", healthConfig.SlowTaskThreshold)
	if len(metrics.SlowTasks) == 0 {
		log.Printf("无慢任务")
	} else {
		log.Printf("发现 %d 个慢任务:", len(metrics.SlowTasks))
		for _, task := range metrics.SlowTasks {
			log.Printf("  - 任务ID: %s, 执行时间: %v, 记录时间: %s",
				task.TaskID, task.ExecutionTime, task.Timestamp.Format("15:04:05"))
		}
	}

	// 10. 打印异常任务分析
	log.Printf("\n--- 异常任务分析 ---")
	if len(metrics.FailedTasksList) == 0 {
		log.Printf("无异常任务")
	} else {
		log.Printf("发现 %d 个异常任务:", len(metrics.FailedTasksList))
		for _, task := range metrics.FailedTasksList {
			log.Printf("  - 任务ID: %s, 错误: %s, 记录时间: %s",
				task.TaskID, task.Error, task.Timestamp.Format("15:04:05"))
		}
	}
	log.Printf("==================================\n")
}

// HealthAnalysisWithPoolWithFuncExample 演示 PoolWithFunc 的健康分析功能
func HealthAnalysisWithPoolWithFuncExample() {
	// 创建带函数的线程池
	wfPool, err := NewPoolWithFunc("health-worker-pool", func(arg interface{}) {
		taskInfo := arg.(map[string]interface{})
		taskName := taskInfo["name"].(string)
		duration := taskInfo["duration"].(time.Duration)

		log.Printf("处理任务: %s", taskName)
		time.Sleep(duration)

		// 模拟部分任务失败
		if taskName == "error-task" {
			panic("模拟任务异常")
		}
	}, 10)
	if err != nil {
		log.Fatalf("创建线程池失败: %v", err)
	}
	defer wfPool.Release()

	// 配置健康分析
	healthConfig := HealthConfig{
		Enabled:              true,
		SlowTaskThreshold:    time.Millisecond * 300,
		MaxSlowTaskRecords:   50,
		MaxFailedTaskRecords: 50,
	}
	wfPool.SetHealthConfig(healthConfig)

	// 提交不同类型的任务
	tasks := []struct {
		id       string
		name     string
		duration time.Duration
		tracked  bool
	}{
		{"task-1", "fast-task", time.Millisecond * 100, true},
		{"task-2", "slow-task", time.Millisecond * 500, true},
		{"task-3", "error-task", time.Millisecond * 50, true},
		{"", "untracked-task", time.Millisecond * 200, false},
	}

	for _, task := range tasks {
		arg := map[string]interface{}{
			"name":     task.name,
			"duration": task.duration,
		}

		if task.tracked {
			wfPool.InvokeWithTaskID(task.id, arg)
		} else {
			wfPool.Invoke(arg)
		}
	}

	// 等待任务完成
	time.Sleep(time.Second * 2)

	// 打印健康分析报告
	metrics := wfPool.Metrics()
	log.Printf("\n========== PoolWithFunc 健康分析报告 ==========")
	log.Printf("被追踪任务数: %d", metrics.TrackedTasks)
	log.Printf("慢任务数: %d", len(metrics.SlowTasks))
	log.Printf("异常任务数: %d", len(metrics.FailedTasksList))

	for _, task := range metrics.SlowTasks {
		log.Printf("  慢任务: %s (耗时: %v)", task.TaskID, task.ExecutionTime)
	}

	for _, task := range metrics.FailedTasksList {
		log.Printf("  异常任务: %s (错误: %s)", task.TaskID, task.Error)
	}
	log.Printf("===============================================\n")
}

// WebServerExample 演示如何在Web服务器上下文中使用antspool
func WebServerExample() {
	// 创建一个专门用于处理HTTP请求的线程池
	reqPool, err := New("http-request-pool", 100) // 100个并发处理线程
	if err != nil {
		log.Fatalf("创建请求线程池失败: %v", err)
	}

	// 模拟HTTP请求处理
	handleRequest := func(requestID, userID string) {
		// 提取需要的变量，避免直接引用HTTP请求对象
		taskParams := struct {
			RequestID string
			UserID    string
		}{RequestID: requestID, UserID: userID}

		if err := reqPool.Submit(func() {
			// 处理业务逻辑
			processRequest(taskParams)
		}); err != nil {
			log.Printf("提交请求 %s 失败: %v", requestID, err)
		}
	}

	// 模拟100个并发请求
	for i := 0; i < 100; i++ {
		handleRequest(fmt.Sprintf("req-%d", i), fmt.Sprintf("user-%d", i%10))
	}

	// 定期打印池状态
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()

		for range ticker.C {
			metrics := reqPool.Metrics()
			log.Printf("请求线程池状态 - 运行协程数: %d, 等待任务数: %d, 总任务数: %d, 失败任务数: %d",
				metrics.RunningGoroutines, metrics.WaitingTasks, metrics.TotalTasks, metrics.FailedTasks)
		}
	}()
}

// processRequest 处理请求的辅助函数
func processRequest(params struct {
	RequestID string
	UserID    string
}) {
	log.Printf("处理请求 %s，用户 %s", params.RequestID, params.UserID)
	time.Sleep(time.Millisecond * 200) // 模拟业务处理时间
}
