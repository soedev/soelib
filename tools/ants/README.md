# Ants 线程池工具包

基于 [ants](https://github.com/panjf2000/ants) 的 Go 协程池封装，提供开箱即用的最佳实践。

## 特性

- ✅ **简单易用** - 全局池和独立池两种使用方式
- ✅ **并发安全** - 提供自动数据拷贝方法，避免竞态条件
- ✅ **类型安全** - 支持泛型，编译时类型检查（Go 1.18+）
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

    // 3. 提交带数据的任务（自动拷贝，并发安全）
    data := map[string]interface{}{
        "orderId": "12345",
        "amount":  100.0,
    }
    ants.SubmitTaskWithData(data, func(d interface{}) {
        // d 是独立副本，并发安全
        dataMap := d.(map[string]interface{})
        fmt.Println(dataMap["orderId"])
    })

    // 4. 提交带类型安全的任务（泛型，Go 1.18+）
    type Order struct {
        ID     string
        Amount float64
    }
    order := Order{ID: "12345", Amount: 100.0}
    ants.SubmitTaskGeneric(order, func(o Order) {
        // o 是独立副本，类型安全
        fmt.Println(o.ID)
    })

    // 5. 提交带 ID 的任务（会被追踪）
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

### 1. 并发安全的数据传递

协程池最常见的问题是并发访问共享数据（如 map、slice）导致的竞态条件。本包提供三种安全的数据传递方式：

#### 方式 1: SubmitTaskWithData（推荐）

自动处理数据拷贝，适用于可 JSON 序列化的数据。

```go
// 传递 map（自动拷贝）
data := map[string]interface{}{
    "userId": 123,
    "action": "login",
    "metadata": map[string]string{
        "ip": "192.168.1.1",
    },
}

err := ants.SubmitTaskWithData(data, func(d interface{}) {
    // d 是独立副本，可以安全使用
    dataMap := d.(map[string]interface{})
    processLogin(dataMap)
})

// 传递 struct
type User struct {
    ID   int
    Name string
}
user := User{ID: 123, Name: "test"}
ants.SubmitTaskWithData(user, func(d interface{}) {
    u := d.(User)
    fmt.Println(u.Name)
})
```

#### 方式 2: SubmitTaskGeneric（类型安全，Go 1.18+）

使用泛型提供类型安全的数据传递。

```go
// 传递 struct（类型安全）
type Order struct {
    ID     string
    Amount float64
    Items  []string
}

order := Order{
    ID:     "ORD-123",
    Amount: 99.99,
    Items:  []string{"item1", "item2"},
}

err := ants.SubmitTaskGeneric(order, func(o Order) {
    // o 是独立副本，类型安全，无需类型断言
    fmt.Printf("处理订单: %s, 金额: %.2f\n", o.ID, o.Amount)
    for _, item := range o.Items {
        processItem(item)
    }
})

// 传递 map（类型安全）
data := map[string]string{
    "key1": "value1",
    "key2": "value2",
}
ants.SubmitTaskGeneric(data, func(d map[string]string) {
    // 类型安全，无需断言
    fmt.Println(d["key1"])
})
```

#### 方式 3: 手动拷贝（最灵活）

对于特殊场景，可以手动序列化数据。

```go
data := map[string]interface{}{"key": "value"}

// 在提交前序列化
dataBytes, _ := json.Marshal(data)

ants.SubmitTask(func() {
    // 在任务中反序列化，得到独立副本
    var d map[string]interface{}
    json.Unmarshal(dataBytes, &d)
    
    // 安全使用
    processData(d)
})
```

#### ⚠️ 常见错误示例

```go
// ❌ 错误：直接引用外部 map
data := map[string]string{"key": "value"}
ants.SubmitTask(func() {
    json.Marshal(data) // 可能并发访问 data，导致 panic
})

// ❌ 错误：在循环中引用循环变量
for _, item := range items {
    ants.SubmitTask(func() {
        process(item) // item 被所有 goroutine 共享
    })
}

// ✅ 正确：使用 SubmitTaskWithData
data := map[string]string{"key": "value"}
ants.SubmitTaskWithData(data, func(d interface{}) {
    dataMap := d.(map[string]string)
    json.Marshal(dataMap) // 安全
})

// ✅ 正确：在循环中使用泛型方法
for _, item := range items {
    ants.SubmitTaskGeneric(item, func(i Item) {
        process(i) // i 是独立副本
    })
}
```

### 2. 健康分析

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

### 3. 自动监控

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

### 4. 上下文支持

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

### 5. PoolWithFunc

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
| `SubmitTask(task func())` | 提交普通任务（不追踪，需手动处理并发安全） |
| `SubmitTaskWithData(data, task)` | 提交带数据的任务（自动拷贝，并发安全）⭐ |
| `SubmitTaskGeneric[T](data, task)` | 提交类型安全的任务（泛型，Go 1.18+）⭐ |
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

### 场景零：并发安全的数据处理（重要）

这是最常见也最容易出错的场景，展示如何安全地处理并发数据。

```go
package main

import (
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/soedev/soelib/tools/ants"
)

// 示例1: 处理订单数据（推荐方式）
func processOrders() {
    // 初始化协程池
    ants.InitCoroutinePool()
    defer ants.CoroutineRelease()
    
    // 订单数据结构
    type Order struct {
        ID       string
        Amount   float64
        Items    []string
        Metadata map[string]interface{}
    }
    
    orders := []Order{
        {
            ID:     "ORD-001",
            Amount: 99.99,
            Items:  []string{"item1", "item2"},
            Metadata: map[string]interface{}{
                "source": "web",
                "user_id": 123,
            },
        },
        // ... 更多订单
    }
    
    // ✅ 方法1: 使用泛型（最推荐）
    for _, order := range orders {
        ants.SubmitTaskGeneric(order, func(o Order) {
            // o 是独立副本，完全安全
            fmt.Printf("处理订单: %s, 金额: %.2f\n", o.ID, o.Amount)
            
            // 可以安全地序列化
            jsonData, _ := json.Marshal(o)
            saveToDatabase(jsonData)
        })
    }
    
    // ✅ 方法2: 使用 SubmitTaskWithData
    for _, order := range orders {
        ants.SubmitTaskWithData(order, func(d interface{}) {
            o := d.(Order)
            processOrder(o)
        })
    }
}

// 示例2: 处理 map 数据
func processMapData() {
    ants.InitCoroutinePool()
    defer ants.CoroutineRelease()
    
    // ❌ 错误方式：直接引用 map
    data := map[string]interface{}{
        "key1": "value1",
        "key2": 123,
    }
    
    // 这样做是危险的！
    // ants.SubmitTask(func() {
    //     json.Marshal(data) // 可能并发访问 data
    // })
    
    // ✅ 正确方式1：使用泛型
    ants.SubmitTaskGeneric(data, func(d map[string]interface{}) {
        // d 是独立副本
        jsonData, _ := json.Marshal(d)
        fmt.Println(string(jsonData))
    })
    
    // ✅ 正确方式2：手动序列化
    dataBytes, _ := json.Marshal(data)
    ants.SubmitTask(func() {
        var d map[string]interface{}
        json.Unmarshal(dataBytes, &d)
        processData(d)
    })
}

// 示例3: 在 HTTP 处理器中使用
func handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
    // 提取请求数据
    type RequestData struct {
        Path    string
        Method  string
        Headers map[string][]string
        Body    []byte
    }
    
    body, _ := io.ReadAll(r.Body)
    requestData := RequestData{
        Path:    r.URL.Path,
        Method:  r.Method,
        Headers: r.Header,
        Body:    body,
    }
    
    // ✅ 使用泛型安全传递
    ants.SubmitTaskGeneric(requestData, func(data RequestData) {
        // 异步处理请求
        processRequest(data)
    })
    
    // 立即返回响应
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "processing",
    })
}

// 示例4: 批量处理数据库记录
func processDatabaseRecords(db *gorm.DB) {
    ants.InitCoroutinePool()
    defer ants.CoroutineRelease()
    
    type User struct {
        ID     uint
        Name   string
        Email  string
        Status string
    }
    
    var users []User
    db.Where("status = ?", "pending").Find(&users)
    
    // ✅ 使用泛型传递用户数据
    for _, user := range users {
        ants.SubmitTaskGeneric(user, func(u User) {
            // u 是独立副本
            // 在任务中处理用户
            processUser(u)
            
            // 更新数据库
            db.Model(&User{}).Where("id = ?", u.ID).
                Update("status", "processed")
        })
    }
}

func saveToDatabase(data []byte) {
    // 保存到数据库
}

func processOrder(order Order) {
    // 处理订单
}

func processData(data map[string]interface{}) {
    // 处理数据
}

func processRequest(data RequestData) {
    // 处理请求
}

func processUser(user User) {
    // 处理用户
}
```

**关键要点：**
1. ✅ 使用 `SubmitTaskGeneric` 或 `SubmitTaskWithData` 自动处理数据拷贝
2. ✅ 避免在闭包中直接引用外部的 map、slice、struct 指针
3. ✅ 在循环中使用泛型方法，避免循环变量共享问题
4. ✅ HTTP 请求处理时，先提取数据再提交任务

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

### 4. 并发安全最佳实践

#### 原则 1: 优先使用 SubmitTaskWithData 或 SubmitTaskGeneric

```go
// ✅ 推荐：使用 SubmitTaskWithData（自动处理并发安全）
data := map[string]interface{}{"key": "value"}
ants.SubmitTaskWithData(data, func(d interface{}) {
    // d 是独立副本，完全安全
    processData(d)
})

// ✅ 推荐：使用 SubmitTaskGeneric（类型安全）
type Task struct {
    ID   string
    Data map[string]string
}
task := Task{ID: "123", Data: map[string]string{"key": "value"}}
ants.SubmitTaskGeneric(task, func(t Task) {
    // t 是独立副本，类型安全
    processTask(t)
})
```

#### 原则 2: 避免闭包陷阱

```go
// ❌ 错误：直接引用循环变量
for i := 0; i < 10; i++ {
    pool.Submit(func() {
        fmt.Println(i) // 可能都打印 10
    })
}

// ✅ 方法1：使用泛型方法
for i := 0; i < 10; i++ {
    ants.SubmitTaskGeneric(i, func(index int) {
        fmt.Println(index) // 正确打印 0-9
    })
}

// ✅ 方法2：使用局部变量
for i := 0; i < 10; i++ {
    index := i
    pool.Submit(func() {
        fmt.Println(index) // 正确打印 0-9
    })
}
```

#### 原则 3: 避免共享可变数据

```go
// ❌ 错误：多个任务共享同一个 map
sharedMap := make(map[string]int)
for i := 0; i < 10; i++ {
    pool.Submit(func() {
        sharedMap["count"]++ // 并发写入，导致竞态条件
    })
}

// ✅ 正确：每个任务使用独立的数据
for i := 0; i < 10; i++ {
    data := map[string]int{"count": i}
    ants.SubmitTaskWithData(data, func(d interface{}) {
        dataMap := d.(map[string]int)
        process(dataMap["count"]) // 安全
    })
}
```

#### 原则 4: HTTP 请求处理

```go
// ❌ 错误：直接引用 Request 对象
func handleRequest(r *http.Request) {
    pool.Submit(func() {
        log.Println(r.URL.Path) // r 可能已被回收或修改
    })
}

// ✅ 方法1：提取需要的数据
func handleRequest(r *http.Request) {
    path := r.URL.Path
    method := r.Method
    pool.Submit(func() {
        log.Printf("%s %s", method, path) // 安全
    })
}

// ✅ 方法2：使用 struct 封装
func handleRequest(r *http.Request) {
    type RequestData struct {
        Path   string
        Method string
        Header map[string][]string
    }
    data := RequestData{
        Path:   r.URL.Path,
        Method: r.Method,
        Header: r.Header,
    }
    ants.SubmitTaskGeneric(data, func(d RequestData) {
        processRequest(d) // 类型安全
    })
}
```

#### 原则 5: 数据库对象处理

```go
// ❌ 错误：在闭包中使用 DB 连接
func processUsers(db *gorm.DB) {
    var users []User
    db.Find(&users)
    
    for _, user := range users {
        pool.Submit(func() {
            // user 被所有 goroutine 共享
            // db 可能被其他 goroutine 使用
            db.Model(&user).Update("status", "processed")
        })
    }
}

// ✅ 正确：传递用户 ID，在任务中重新查询
func processUsers(db *gorm.DB) {
    var users []User
    db.Find(&users)
    
    for _, user := range users {
        userID := user.ID
        pool.Submit(func() {
            // 在任务中创建新的 DB 会话
            var u User
            db.First(&u, userID)
            db.Model(&u).Update("status", "processed")
        })
    }
}

// ✅ 更好：使用泛型传递完整数据
func processUsers(db *gorm.DB) {
    var users []User
    db.Find(&users)
    
    for _, user := range users {
        ants.SubmitTaskGeneric(user, func(u User) {
            // u 是独立副本
            db.Model(&u).Update("status", "processed")
        })
    }
}
```

### 5. 选择合适的方法

| 场景 | 推荐方法 | 原因 |
|------|---------|------|
| 传递 struct/map/slice | `SubmitTaskGeneric` | 类型安全，自动拷贝 |
| 传递复杂数据结构 | `SubmitTaskWithData` | 自动拷贝，灵活 |
| 传递简单值（string/int） | `SubmitTask` | 性能最优 |
| 需要追踪的任务 | `SubmitTaskWithID` | 便于监控 |
| 需要取消的任务 | `SubmitWithContext` | 支持取消 |

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

### 并发安全

1. ✅ **优先使用** `SubmitTaskWithData` 或 `SubmitTaskGeneric` 传递数据
2. ✅ **避免**在闭包中直接引用外部的 map、slice、指针
3. ✅ **避免**在循环中直接引用循环变量
4. ✅ **避免**多个任务共享可变数据
5. ✅ 传递数据的副本而非引用

### 资源管理

6. ✅ 总是使用 `defer pool.Release()` 确保资源释放
7. ✅ 避免在任务中执行长时间阻塞操作
8. ✅ 定期监控池状态，及时调整大小

### 性能优化

9. ✅ 使用 `SubmitWithContext` 处理可取消任务
10. ✅ 健康分析默认关闭，不影响性能
11. ✅ 只对关键任务使用 `SubmitTaskWithID`
12. ✅ 自定义打印函数集成自己的日志框架

### 方法选择指南

```go
// 场景1: 传递复杂数据（map、struct、slice）
✅ 使用 SubmitTaskGeneric 或 SubmitTaskWithData

// 场景2: 传递简单值（string、int、bool）
✅ 使用 SubmitTask（直接捕获值）

// 场景3: 需要追踪和监控
✅ 使用 SubmitTaskWithID

// 场景4: 需要取消或超时控制
✅ 使用 SubmitWithContext 或 SubmitWithTimeout
```

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
