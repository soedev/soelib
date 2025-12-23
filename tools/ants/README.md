# Ants 线程池工具包

基于 [ants](https://github.com/panjf2000/ants) 的 Go 协程池封装，提供开箱即用的最佳实践。

## 特性

- ✅ **简单易用** - 全局池和独立池两种使用方式
- ✅ **健康分析** - 自动追踪慢任务和异常任务
- ✅ **自动监控** - 可配置的自动监控打印
- ✅ **Panic 处理** - 自动捕获任务 panic，不影响池运行
- ✅ **上下文支持** - 支持 context 取消和超时
- ✅ **完善监控** - 丰富的性能指标
- ✅ **零性能损耗** - 健康分析仅对带 ID 任务生效

## 快速开始

### 方式一：使用全局池（推荐）

```go
package main

import (
    "time"
    "github.com/soedev/soelib/tools/ants"
)

func main() {
    // 1. 初始化全局池
    config := ants.GlobalPoolConfig{
        Name:     "my-app-pool",
        Size:     1000,
        PreAlloc: true,
        Health: ants.HealthConfig{
            Enabled:              true,
            SlowTaskThreshold:    time.Second * 5,
            MaxSlowTaskRecords:   100,
            MaxFailedTaskRecords: 100,
        },
        Monitor: ants.MonitorConfig{
            Enabled:             true,
            Interval:            time.Minute * 5,
            PrintBasicMetrics:   true,
            PrintHealthInfo:     true,
            MaxSlowTasksPrint:   5,
            MaxFailedTasksPrint: 5,
        },
    }
    ants.InitCoroutinePoolWithConfig(config)
    defer ants.CoroutineRelease()

    // 2. 提交普通任务（不追踪）
    ants.SubmitTask(func() {
        // 执行任务
    })

    // 3. 提交带 ID 的任务（会被追踪）
    ants.SubmitTaskWithID("order-12345", func() {
        // 执行订单处理
    })
}
```

### 方式二：创建独立池

```go
package main

import (
    "github.com/soedev/soelib/tools/ants"
)

func main() {
    // 创建独立池
    pool, err := ants.New("worker-pool", 100)
    if err != nil {
        panic(err)
    }
    defer pool.Release()

    // 提交任务
    pool.Submit(func() {
        // 执行任务
    })
}
```

## 核心功能

### 1. 健康分析

自动追踪慢任务和异常任务，帮助发现性能瓶颈和问题。

```go
// 启用健康分析
healthConfig := ants.HealthConfig{
    Enabled:              true,
    SlowTaskThreshold:    time.Second * 2,  // 超过2秒视为慢任务
    MaxSlowTaskRecords:   100,
    MaxFailedTaskRecords: 100,
}
pool.SetHealthConfig(healthConfig)

// 提交带 ID 的任务（会被追踪）
pool.SubmitWithTaskID("user-login-123", func() {
    // 执行登录逻辑
})

// 获取健康分析报告
metrics := pool.Metrics()
fmt.Printf("慢任务数: %d\n", len(metrics.SlowTasks))
fmt.Printf("异常任务数: %d\n", len(metrics.FailedTasksList))

// 打印慢任务详情
for _, task := range metrics.SlowTasks {
    fmt.Printf("慢任务: %s, 执行时间: %v\n", task.TaskID, task.ExecutionTime)
}

// 打印异常任务详情
for _, task := range metrics.FailedTasksList {
    fmt.Printf("异常任务: %s, 错误: %s\n", task.TaskID, task.Error)
}
```

**注意**：
- 只有通过 `SubmitWithTaskID` 提交的任务才会被追踪
- 普通的 `Submit` 方法不受影响，零性能损耗
- 建议：80% 普通任务 + 20% 关键任务（带ID）

### 2. 自动监控

配置自动监控，定时打印池状态和健康分析。

```go
config := ants.GlobalPoolConfig{
    Name: "my-pool",
    Size: 1000,
    Monitor: ants.MonitorConfig{
        Enabled:             true,              // 启用自动监控
        Interval:            time.Minute * 5,   // 每5分钟打印一次
        PrintBasicMetrics:   true,              // 打印基础指标
        PrintHealthInfo:     true,              // 打印健康分析
        MaxSlowTasksPrint:   5,                 // 最多打印5个慢任务
        MaxFailedTasksPrint: 5,                 // 最多打印5个异常任务
    },
}
ants.InitCoroutinePoolWithConfig(config)
defer ants.CoroutineRelease() // 会自动停止监控
```

**自定义打印函数**（集成自己的日志框架）：

```go
config.Monitor.CustomPrinter = func(metrics ants.Metrics) {
    // 使用您的日志框架
    logger.Info("线程池监控",
        "pool", metrics.Name,
        "running", metrics.RunningGoroutines,
        "waiting", metrics.WaitingTasks,
        "total", metrics.TotalTasks,
        "success", metrics.SuccessTasks,
        "failed", metrics.FailedTasks,
        "slow_tasks", len(metrics.SlowTasks),
    )
}
```

**动态控制监控**：

```go
// 启动监控
ants.StartGlobalMonitor(monitorConfig)

// 停止监控
ants.StopGlobalMonitor()

// 检查监控状态
if ants.IsMonitorRunning() {
    fmt.Println("监控正在运行")
}
```

### 3. 上下文支持

支持 context 取消和超时控制。

```go
// 带上下文的任务
ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
defer cancel()

pool.SubmitWithContext(ctx, func(ctx context.Context) {
    select {
    case <-ctx.Done():
        return // 任务被取消
    default:
        // 执行任务
    }
})

// 带超时的任务
pool.SubmitWithTimeout(time.Second*2, func() {
    // 执行任务
})
```

### 4. PoolWithFunc

适用于所有任务执行相同逻辑的场景。

```go
// 创建带函数的池
pool, err := ants.NewPoolWithFunc("worker-pool", func(arg interface{}) {
    // 处理任务
    data := arg.(MyData)
    processData(data)
}, 100)
if err != nil {
    panic(err)
}
defer pool.Release()

// 提交任务
pool.Invoke(myData)

// 提交带 ID 的任务（会被追踪）
pool.InvokeWithTaskID("task-123", myData)
```

## API 参考

### 全局池方法

| 方法 | 描述 |
|------|------|
| `InitCoroutinePool()` | 使用默认配置初始化全局池 |
| `InitCoroutinePoolWithConfig(config)` | 使用自定义配置初始化全局池 |
| `SubmitTask(task func())` | 提交普通任务（不追踪） |
| `SubmitTaskWithID(taskID, task)` | 提交带 ID 的任务（会被追踪） |
| `SetGlobalHealthConfig(config)` | 设置健康分析配置 |
| `GetGlobalHealthConfig()` | 获取健康分析配置 |
| `GetGlobalMetrics()` | 获取监控指标 |
| `StartGlobalMonitor(config)` | 启动自动监控 |
| `StopGlobalMonitor()` | 停止自动监控 |
| `IsMonitorRunning()` | 检查监控是否运行 |
| `CoroutineRelease()` | 释放全局池资源 |
| `GetPool()` | 获取全局池实例 |

### Pool 方法

| 方法 | 描述 |
|------|------|
| `New(name, size)` | 创建新池 |
| `Submit(task)` | 提交普通任务 |
| `SubmitWithTaskID(taskID, task)` | 提交带 ID 的任务 |
| `SubmitWithContext(ctx, task)` | 提交带上下文的任务 |
| `SubmitWithTimeout(timeout, task)` | 提交带超时的任务 |
| `SetHealthConfig(config)` | 设置健康分析配置 |
| `GetHealthConfig()` | 获取健康分析配置 |
| `Metrics()` | 获取监控指标 |
| `Running()` | 获取运行中的协程数 |
| `Waiting()` | 获取等待中的任务数 |
| `Tune(size)` | 动态调整池大小 |
| `Release()` | 释放池资源 |

### PoolWithFunc 方法

| 方法 | 描述 |
|------|------|
| `NewPoolWithFunc(name, func, size)` | 创建带函数的池 |
| `Invoke(arg)` | 提交任务 |
| `InvokeWithTaskID(taskID, arg)` | 提交带 ID 的任务 |
| `SetHealthConfig(config)` | 设置健康分析配置 |
| `Metrics()` | 获取监控指标 |
| `Release()` | 释放池资源 |

### 配置结构

#### GlobalPoolConfig

```go
type GlobalPoolConfig struct {
    Name     string        // 线程池名称
    Size     int           // 线程池大小
    PreAlloc bool          // 是否预分配
    Health   HealthConfig  // 健康分析配置
    Monitor  MonitorConfig // 监控配置
}
```

#### HealthConfig

```go
type HealthConfig struct {
    Enabled              bool          // 是否启用健康分析
    SlowTaskThreshold    time.Duration // 慢任务阈值
    MaxSlowTaskRecords   int           // 最大慢任务记录数
    MaxFailedTaskRecords int           // 最大异常任务记录数
}
```

#### MonitorConfig

```go
type MonitorConfig struct {
    Enabled             bool                  // 是否启用自动监控
    Interval            time.Duration         // 监控打印间隔
    PrintBasicMetrics   bool                  // 是否打印基础指标
    PrintHealthInfo     bool                  // 是否打印健康分析
    MaxSlowTasksPrint   int                   // 最多打印多少个慢任务
    MaxFailedTasksPrint int                   // 最多打印多少个异常任务
    CustomPrinter       func(metrics Metrics) // 自定义打印函数
}
```

#### Metrics

```go
type Metrics struct {
    Name                 string              // 线程池名称
    RunningGoroutines    int                 // 运行中的协程数
    WaitingTasks         int                 // 等待中的任务数
    TotalTasks           int64               // 总任务数
    SuccessTasks         int64               // 成功任务数
    FailedTasks          int64               // 失败任务数
    TrackedTasks         int64               // 被追踪的任务数
    TaskExecutionTime    time.Duration       // 平均执行时间
    MaxTaskExecutionTime time.Duration       // 最大执行时间
    MinTaskExecutionTime time.Duration       // 最小执行时间
    SlowTasks            []SlowTaskRecord    // 慢任务列表
    FailedTasksList      []FailedTaskRecord  // 异常任务列表
    PoolCapacity         int                 // 池容量
    Timestamp            time.Time           // 指标生成时间
}
```

## 使用场景

### 场景一：Web 应用

```go
func main() {
    // 初始化全局池（启用健康分析和自动监控）
    config := ants.GlobalPoolConfig{
        Name:     "web-app-pool",
        Size:     5000,
        PreAlloc: true,
        Health: ants.HealthConfig{
            Enabled:              true,
            SlowTaskThreshold:    time.Second * 3,
            MaxSlowTaskRecords:   100,
            MaxFailedTaskRecords: 100,
        },
        Monitor: ants.MonitorConfig{
            Enabled:   true,
            Interval:  time.Minute * 5,
            CustomPrinter: func(metrics ants.Metrics) {
                // 集成您的日志框架
                logger.Info("线程池监控", "metrics", metrics)
            },
        },
    }
    ants.InitCoroutinePoolWithConfig(config)
    defer ants.CoroutineRelease()

    // HTTP 处理器
    http.HandleFunc("/api/process", func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        
        // 提交带 ID 的任务进行追踪
        ants.SubmitTaskWithID(requestID, func() {
            processRequest(requestID)
        })
        
        w.WriteHeader(http.StatusAccepted)
    })

    http.ListenAndServe(":8080", nil)
}
```

### 场景二：数据处理

```go
func main() {
    // 创建专用池
    pool, _ := ants.New("data-processor", 100)
    defer pool.Release()

    // 启用健康分析
    pool.SetHealthConfig(ants.HealthConfig{
        Enabled:           true,
        SlowTaskThreshold: time.Second * 10,
    })

    // 批量处理数据
    for _, item := range dataList {
        itemCopy := item // 避免闭包陷阱
        pool.SubmitWithTaskID(item.ID, func() {
            processData(itemCopy)
        })
    }

    // 等待完成
    time.Sleep(time.Minute)

    // 查看健康报告
    metrics := pool.Metrics()
    fmt.Printf("处理完成: 总任务=%d, 成功=%d, 失败=%d, 慢任务=%d\n",
        metrics.TotalTasks, metrics.SuccessTasks, 
        metrics.FailedTasks, len(metrics.SlowTasks))
}
```

## 最佳实践

### 1. 线程池大小设置

- **CPU 密集型**：`runtime.NumCPU()`
- **IO 密集型**：`runtime.NumCPU() * 2~4`
- **混合型**：根据实际测试调整

### 2. 任务追踪策略

- 普通任务：使用 `Submit()`，性能最优
- 关键任务：使用 `SubmitWithTaskID()`，便于追踪
- 建议比例：80% 普通任务 + 20% 关键任务

### 3. 监控配置

**生产环境**：
```go
Monitor: ants.MonitorConfig{
    Enabled:   true,
    Interval:  time.Minute * 10,  // 10分钟打印一次
}
```

**开发环境**：
```go
Monitor: ants.MonitorConfig{
    Enabled:   true,
    Interval:  time.Second * 30,  // 30秒打印一次
}
```

### 4. 避免闭包陷阱

```go
// ❌ 错误：直接引用循环变量
for i := 0; i < 10; i++ {
    pool.Submit(func() {
        fmt.Println(i) // 可能都打印 10
    })
}

// ✅ 正确：使用局部变量
for i := 0; i < 10; i++ {
    index := i
    pool.Submit(func() {
        fmt.Println(index) // 正确打印 0-9
    })
}
```

### 5. HTTP 请求处理

```go
// ❌ 错误：直接引用 Request 对象
func handleRequest(r *http.Request) {
    pool.Submit(func() {
        log.Println(r.URL.Path) // r 可能已被回收
    })
}

// ✅ 正确：传递变量副本
func handleRequest(r *http.Request) {
    path := r.URL.Path
    pool.Submit(func() {
        log.Println(path) // 安全
    })
}
```

## 性能优化

### 1. 预分配内存

```go
config := ants.GlobalPoolConfig{
    PreAlloc: true,  // 启用预分配，减少运行时内存分配
}
```

### 2. 合理设置记录数量

```go
Health: ants.HealthConfig{
    MaxSlowTaskRecords:   100,  // 根据实际需求调整
    MaxFailedTaskRecords: 100,
}
```

### 3. 动态调整池大小

```go
// 根据负载动态调整
metrics := pool.Metrics()
if float64(metrics.WaitingTasks) / float64(metrics.PoolCapacity) > 0.8 {
    pool.Tune(metrics.PoolCapacity * 2)  // 增大池容量
}
```

## 注意事项

1. ✅ 总是使用 `defer pool.Release()` 确保资源释放
2. ✅ 避免在任务中执行长时间阻塞操作
3. ✅ 定期监控池状态，及时调整大小
4. ✅ 传递变量副本而非引用
5. ✅ 使用 `SubmitWithContext` 处理可取消任务
6. ✅ 健康分析默认关闭，不影响性能
7. ✅ 只对关键任务使用 `SubmitWithTaskID`
8. ✅ 自定义打印函数集成自己的日志框架

## 故障排查

### 问题：协程池未初始化

**错误信息**：
```
[ants] 协程池未初始化，无法提交任务
```

**解决方案**：
```go
ants.InitCoroutinePool()  // 先初始化
```

### 问题：慢任务过多

**解决方案**：
1. 检查慢任务阈值是否合理
2. 优化任务逻辑
3. 增加池大小

### 问题：监控没有输出

**解决方案**：
```go
// 检查监控状态
if !ants.IsMonitorRunning() {
    ants.StartGlobalMonitor(config)
}
```

## 示例代码

查看 `example.go` 获取完整示例：
- 基本使用
- 健康分析
- 自动监控
- PoolWithFunc
- Web 应用场景

## 测试

运行测试：
```bash
go test -v
```

运行特定测试：
```bash
go test -v -run TestHealthAnalysis
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
