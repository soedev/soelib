# antspool - 最佳实践的 ants 线程池封装

## 简介

antspool 是一个基于 ants 线程池库的封装，集成了最佳实践，旨在简化线程池的使用并避免常见错误。它提供了：

- 开箱即用的默认配置
- 安全的任务提交机制
- 内置的 panic 处理
- 上下文感知的任务执行
- 完善的监控指标
- 优雅的关闭机制

## 安装

将 antspool 目录拷贝到你的项目中，然后在代码中导入：

```go
import "your-project/ants"
```

## 快速开始

### 1. 创建线程池

```go
// 创建默认配置的线程池
pool, err := antspool.New("my-pool")
if err != nil {
    log.Fatalf("Failed to create pool: %v", err)
}
defer pool.Release()

// 或者指定线程池大小
pool, err := antspool.New("my-pool", 100) // 100个工作协程
```

### 2. 提交任务

```go
// 基本任务提交
for i := 0; i < 10; i++ {
    taskID := i // 注意：使用局部变量副本避免闭包陷阱
    if err := pool.Submit(func() {
        log.Printf("Processing task %d", taskID)
        // 任务逻辑
    }); err != nil {
        log.Printf("Failed to submit task: %v", err)
    }
}
```

### 3. 上下文感知任务

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
defer cancel()

if err := pool.SubmitWithContext(ctx, func(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return // 任务被取消
        default:
            // 执行任务逻辑
        }
    }
}); err != nil {
    log.Printf("Failed to submit task: %v", err)
}
```

### 4. 带超时的任务

```go
if err := pool.SubmitWithTimeout(time.Second*2, func() {
    // 可能超时的任务
    time.Sleep(time.Millisecond * 3000)
}); err != nil {
    log.Printf("Failed to submit task: %v", err)
}
```

### 5. 使用 PoolWithFunc

```go
// 创建带函数的线程池
wfPool, err := antspool.NewPoolWithFunc("worker-pool", func(arg interface{}) {
    log.Printf("Processing: %v", arg)
    // 任务逻辑
})
if err != nil {
    log.Fatalf("Failed to create PoolWithFunc: %v", err)
}
defer wfPool.Release()

// 提交任务
for i := 0; i < 5; i++ {
    if err := wfPool.Invoke(i); err != nil {
        log.Printf("Failed to invoke task: %v", err)
    }
}
```

## 核心特性

### 1. 安全的闭包处理

自动处理闭包变量捕获问题，避免常见的 "循环变量陷阱"。

### 2. 内置的 Panic 处理

所有任务中的 panic 都会被捕获并记录，不会导致整个线程池崩溃。

### 3. 上下文感知

支持通过 context 控制任务的生命周期，实现优雅的取消和超时。

### 4. 完善的监控

提供详细的监控指标，包括：
- 运行中的协程数
- 等待中的任务数
- 总任务数
- 失败任务数

```go
// 获取监控指标
metrics := pool.Metrics()
log.Printf("Pool status: %+v", metrics)
```

### 5. 优雅的关闭

支持带超时的关闭机制，确保所有任务都能正确完成或超时。

```go
// 优雅关闭，等待最多5秒
if err := pool.Release(); err != nil {
    log.Printf("Failed to release pool: %v", err)
}
```

## 最佳实践

### 1. 避免直接引用外部变量

```go
// 错误：直接引用外部变量
func handleRequest(r *http.Request) {
    pool.Submit(func() {
        // 危险：r可能已被回收
        log.Println(r.URL.Path)
    })
}

// 正确：传递变量副本
func handleRequest(r *http.Request) {
    path := r.URL.Path
    pool.Submit(func() {
        log.Println(path) // 使用副本，安全
    })
}
```

### 2. 使用合适的线程池大小

- CPU 密集型任务：设置为 CPU 核心数
- IO 密集型任务：设置为 CPU 核心数的 2-4 倍

### 3. 监控线程池状态

定期监控线程池状态，以便及时调整大小或发现问题：

```go
ticker := time.NewTicker(time.Second * 10)
defer ticker.Stop()

for range ticker.C {
    metrics := pool.Metrics()
    log.Printf("Pool %s: Running=%d, Waiting=%d, Total=%d, Failed=%d",
        metrics.Name, metrics.RunningGoroutines, metrics.WaitingTasks,
        metrics.TotalTasks, metrics.FailedTasks)
}
```

### 4. 动态调整线程池大小

根据负载动态调整线程池大小：

```go
// 根据系统负载调整
if systemLoad > 0.8 {
    pool.Tune(pool.Cap() * 2) // 负载高，增大池容量
} else if systemLoad < 0.3 {
    pool.Tune(pool.Cap() / 2) // 负载低，减小池容量
}
```

## 适用场景

- 高并发 Web 服务器
- 大规模数据处理
- 异步任务队列
- 定时任务执行
- 网络爬虫

## 不适用场景

- 长时间运行的单个任务
- 对延迟要求极高的任务
- 任务间有强依赖关系的场景

## API 参考

### Pool 结构体

| 方法 | 描述 |
|------|------|
| `New(name string, size ...int) (*Pool, error)` | 创建新的线程池 |
| `Submit(task func()) error` | 提交基本任务 |
| `SubmitWithContext(ctx context.Context, task func(ctx context.Context)) error` | 提交带上下文的任务 |
| `SubmitWithTimeout(timeout time.Duration, task func()) error` | 提交带超时的任务 |
| `Tune(size int)` | 动态调整池大小 |
| `Running() int` | 获取运行中的协程数 |
| `Waiting() int` | 获取等待中的任务数 |
| `Metrics() Metrics` | 获取监控指标 |
| `Release() error` | 优雅关闭线程池 |

### PoolWithFunc 结构体

| 方法 | 描述 |
|------|------|
| `NewPoolWithFunc(name string, workerFunc func(interface{}), size ...int) (*PoolWithFunc, error)` | 创建带函数的线程池 |
| `Invoke(arg interface{}) error` | 提交任务 |
| `Tune(size int)` | 动态调整池大小 |
| `Running() int` | 获取运行中的协程数 |
| `Waiting() int` | 获取等待中的任务数 |
| `Metrics() Metrics` | 获取监控指标 |
| `Release() error` | 优雅关闭线程池 |

### Metrics 结构体

| 字段 | 类型 | 描述 |
|------|------|------|
| `Name` | `string` | 线程池名称 |
| `RunningGoroutines` | `int` | 运行中的协程数 |
| `WaitingTasks` | `int` | 等待中的任务数 |
| `TotalTasks` | `int64` | 总任务数 |
| `FailedTasks` | `int64` | 失败任务数 |
| `Timestamp` | `time.Time` | 指标生成时间 |

## 示例

查看 `example.go` 文件获取完整示例：

- 基本使用示例
- 上下文感知任务
- 超时任务
- PoolWithFunc 使用
- Web 服务器场景

## 注意事项

1. 总是使用 `defer pool.Release()` 确保线程池正确关闭
2. 避免在任务中执行长时间阻塞的操作
3. 定期监控线程池状态，及时调整大小
4. 对于 HTTP 请求处理，总是传递变量副本而非请求对象本身
5. 使用 `SubmitWithContext` 处理长时间运行的任务，以便能够取消

## 许可证

MIT
