package ants

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/uber/jaeger-client-go/crossdock/log"
)

// PoolError 定义自定义错误类型，用于提供更详细的错误信息
type PoolError struct {
	Code    int    // 错误码
	Message string // 错误消息
	Err     error  // 原始错误
}

// Error 实现 error 接口
func (e *PoolError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap 实现 errors.Unwrap 接口，用于错误链处理
func (e *PoolError) Unwrap() error {
	return e.Err
}

// 错误码定义
const (
	ErrCodeSubmitTask      = 1001 // 提交任务失败
	ErrCodeCreatePool      = 1002 // 创建池失败
	ErrCodeInvalidConfig   = 1003 // 无效配置
	ErrCodePoolClosed      = 1004 // 池已关闭
	ErrCodeTaskPanic       = 1005 // 任务 panic
	ErrCodeTaskTimeout     = 1006 // 任务超时
	ErrCodeContextCanceled = 1007 // 上下文已取消
)

// ErrorCallback 定义错误回调函数类型
type ErrorCallback func(err *PoolError)

// Pool 是对 ants.Pool 的封装，内置最佳实践
type Pool struct {
	pool           *ants.Pool
	name           string
	totalTasks     int64
	failedTasks    int64
	errorCallbacks []ErrorCallback // 错误回调函数列表
	// 新增字段用于跟踪监控指标
	taskExecutionTimes   []time.Duration // 任务执行时间切片
	maxTaskExecutionTime time.Duration   // 最大任务执行时间
	minTaskExecutionTime time.Duration   // 最小任务执行时间
	goroutinesCreated    int64           // 已创建的协程总数
	goroutinesDestroyed  int64           // 已销毁的协程总数
	lastQueueLength      int             // 上一次的队列长度
	mu                   sync.RWMutex    // 读写锁，保护共享字段
	// 健康分析相关字段
	healthConfig      HealthConfig       // 健康分析配置
	trackedTasks      int64              // 被追踪的任务总数
	slowTaskRecords   []SlowTaskRecord   // 慢任务记录（环形缓冲）
	failedTaskRecords []FailedTaskRecord // 异常任务记录（环形缓冲）
}

// New 创建一个新的线程池，带有默认配置
func New(name string, size ...int) (*Pool, error) {
	if len(size) > 0 {
		return NewWithOptions(name, size[0])
	}
	return NewWithOptions(name, 0)
}

// NewWithOptions 创建一个新的线程池，支持自定义配置选项
func NewWithOptions(name string, size int, optionFuncs ...func(*ants.Options)) (*Pool, error) {
	poolSize := runtime.NumCPU() * 2 // 默认设置为CPU核心数的2倍，适合IO密集型任务
	if size > 0 {
		poolSize = size
	}

	// 先创建池实例
	pool := &Pool{
		name:              name,
		healthConfig:      DefaultHealthConfig(), // 使用默认健康分析配置
		slowTaskRecords:   make([]SlowTaskRecord, 0),
		failedTaskRecords: make([]FailedTaskRecord, 0),
	}

	// 默认配置
	defaultOptions := []ants.Option{
		ants.WithExpiryDuration(time.Minute),
		ants.WithMaxBlockingTasks(poolSize * 3), // 默认任务队列大小设为池大小的3倍
		ants.WithPanicHandler(func(i interface{}) {
			log.Printf("[ants][%s] 捕获到panic: %v", name, i)
			atomic.AddInt64(&pool.failedTasks, 1)
			// 调用错误回调函数 - 注意：这里不需要获取锁，因为callErrorCallbacks已经处理了并发安全
			pool.callErrorCallbacks(&PoolError{
				Code:    ErrCodeTaskPanic,
				Message: "任务执行过程中发生panic",
				Err:     fmt.Errorf("%v", i),
			})
		}),
	}

	// 应用默认选项
	options := ants.Options{}
	for _, opt := range defaultOptions {
		opt(&options)
	}

	// 应用用户自定义选项
	for _, optFunc := range optionFuncs {
		optFunc(&options)
	}

	// 转换为 ants.Option 切片
	optionSlice := []ants.Option{}
	if options.ExpiryDuration > 0 {
		optionSlice = append(optionSlice, ants.WithExpiryDuration(options.ExpiryDuration))
	}
	if options.MaxBlockingTasks > 0 {
		optionSlice = append(optionSlice, ants.WithMaxBlockingTasks(options.MaxBlockingTasks))
	}
	if options.PanicHandler != nil {
		optionSlice = append(optionSlice, ants.WithPanicHandler(options.PanicHandler))
	}
	if options.PreAlloc {
		optionSlice = append(optionSlice, ants.WithPreAlloc(options.PreAlloc))
	}
	if options.Nonblocking {
		optionSlice = append(optionSlice, ants.WithNonblocking(options.Nonblocking))
	}
	if options.Logger != nil {
		optionSlice = append(optionSlice, ants.WithLogger(options.Logger))
	}

	antsPool, err := ants.NewPool(poolSize, optionSlice...)
	if err != nil {
		poolErr := &PoolError{
			Code:    ErrCodeCreatePool,
			Message: "创建线程池失败",
			Err:     err,
		}
		return nil, poolErr
	}

	pool.pool = antsPool
	// 初始化时记录创建的协程数
	atomic.AddInt64(&pool.goroutinesCreated, int64(poolSize))

	return pool, nil
}

// SetHealthConfig 设置健康分析配置
func (p *Pool) SetHealthConfig(config HealthConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthConfig = config
}

// GetHealthConfig 获取健康分析配置
func (p *Pool) GetHealthConfig() HealthConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.healthConfig
}

// SubmitWithTaskID 提交带任务ID的任务到线程池（会被健康分析追踪）
func (p *Pool) SubmitWithTaskID(taskID string, task func()) error {
	atomic.AddInt64(&p.totalTasks, 1)

	// 检查是否启用健康分析
	p.mu.RLock()
	healthEnabled := p.healthConfig.Enabled
	slowThreshold := p.healthConfig.SlowTaskThreshold
	p.mu.RUnlock()

	if healthEnabled {
		atomic.AddInt64(&p.trackedTasks, 1)
	}

	// 包装任务，添加健康分析追踪
	wrappedTask := func() {
		startTime := time.Now()
		var panicErr interface{}
		var taskErr error

		// 执行任务并捕获panic
		defer func() {
			panicErr = recover()
			executionTime := time.Since(startTime)

			// 只有启用健康分析时才记录
			if healthEnabled {
				p.mu.Lock()

				// 检查是否为慢任务
				if slowThreshold > 0 && executionTime > slowThreshold {
					record := SlowTaskRecord{
						TaskID:        taskID,
						ExecutionTime: executionTime,
						Timestamp:     time.Now(),
					}
					p.slowTaskRecords = append(p.slowTaskRecords, record)

					// 限制记录数量
					maxRecords := p.healthConfig.MaxSlowTaskRecords
					if len(p.slowTaskRecords) > maxRecords {
						p.slowTaskRecords = p.slowTaskRecords[len(p.slowTaskRecords)-maxRecords:]
					}
				}

				// 记录异常任务
				if panicErr != nil || taskErr != nil {
					errMsg := ""
					if panicErr != nil {
						errMsg = fmt.Sprintf("panic: %v", panicErr)
					} else {
						errMsg = taskErr.Error()
					}

					record := FailedTaskRecord{
						TaskID:    taskID,
						Error:     errMsg,
						Timestamp: time.Now(),
					}
					p.failedTaskRecords = append(p.failedTaskRecords, record)

					// 限制记录数量
					maxRecords := p.healthConfig.MaxFailedTaskRecords
					if len(p.failedTaskRecords) > maxRecords {
						p.failedTaskRecords = p.failedTaskRecords[len(p.failedTaskRecords)-maxRecords:]
					}
				}

				p.mu.Unlock()
			}

			// 检查是否有 panic
			if panicErr != nil {
				log.Printf("[ants][%s] 任务 %s 捕获到panic: %v", p.name, taskID, panicErr)
				atomic.AddInt64(&p.failedTasks, 1)
				// 调用错误回调函数
				p.callErrorCallbacks(&PoolError{
					Code:    ErrCodeTaskPanic,
					Message: fmt.Sprintf("任务 %s 执行过程中发生panic", taskID),
					Err:     fmt.Errorf("%v", panicErr),
				})
			}
		}()

		// 执行任务
		task()
	}

	err := p.pool.Submit(wrappedTask)
	if err != nil {
		atomic.AddInt64(&p.failedTasks, 1)
		poolErr := &PoolError{
			Code:    ErrCodeSubmitTask,
			Message: fmt.Sprintf("提交任务 %s 失败", taskID),
			Err:     err,
		}
		// 调用错误回调函数
		p.callErrorCallbacks(poolErr)
		return poolErr
	}

	return nil
}

// Submit 提交任务到线程池
func (p *Pool) Submit(task func()) error {
	atomic.AddInt64(&p.totalTasks, 1)

	// 包装任务，添加执行时间跟踪
	wrappedTask := func() {
		startTime := time.Now()
		var panicErr interface{}

		// 执行任务并捕获panic
		defer func() {
			panicErr = recover()
			// 计算任务执行时间
			executionTime := time.Since(startTime)

			// 更新监控指标
			p.mu.Lock()
			// 记录任务执行时间
			p.taskExecutionTimes = append(p.taskExecutionTimes, executionTime)
			// 限制切片长度，避免内存占用过高
			if len(p.taskExecutionTimes) > 1000 {
				p.taskExecutionTimes = p.taskExecutionTimes[len(p.taskExecutionTimes)-1000:]
			}

			// 更新最大和最小执行时间
			if executionTime > p.maxTaskExecutionTime {
				p.maxTaskExecutionTime = executionTime
			}
			if p.minTaskExecutionTime == 0 || executionTime < p.minTaskExecutionTime {
				p.minTaskExecutionTime = executionTime
			}
			p.mu.Unlock()

			// 检查是否有 panic，在锁外处理，避免死锁
			if panicErr != nil {
				log.Printf("[ants][%s] 捕获到panic: %v", p.name, panicErr)
				atomic.AddInt64(&p.failedTasks, 1)
				// 调用错误回调函数
				p.callErrorCallbacks(&PoolError{
					Code:    ErrCodeTaskPanic,
					Message: "任务执行过程中发生panic",
					Err:     fmt.Errorf("%v", panicErr),
				})
			}
		}()

		// 执行任务
		task()
	}

	err := p.pool.Submit(wrappedTask)
	if err != nil {
		atomic.AddInt64(&p.failedTasks, 1)
		poolErr := &PoolError{
			Code:    ErrCodeSubmitTask,
			Message: "提交任务失败",
			Err:     err,
		}
		// 调用错误回调函数
		p.callErrorCallbacks(poolErr)
		return poolErr
	}

	return nil
}

// SubmitWithContext 提交带有上下文的任务
func (p *Pool) SubmitWithContext(ctx context.Context, task func(ctx context.Context)) error {
	atomic.AddInt64(&p.totalTasks, 1)

	// 包装任务，添加执行时间跟踪
	wrappedTask := func() {
		startTime := time.Now()
		var panicErr interface{}
		var ctxCanceled bool

		// 执行任务并捕获panic
		defer func() {
			panicErr = recover()
			// 计算任务执行时间
			executionTime := time.Since(startTime)

			// 更新监控指标
			p.mu.Lock()
			// 记录任务执行时间
			p.taskExecutionTimes = append(p.taskExecutionTimes, executionTime)
			// 限制切片长度，避免内存占用过高
			if len(p.taskExecutionTimes) > 1000 {
				p.taskExecutionTimes = p.taskExecutionTimes[len(p.taskExecutionTimes)-1000:]
			}

			// 更新最大和最小执行时间
			if executionTime > p.maxTaskExecutionTime {
				p.maxTaskExecutionTime = executionTime
			}
			if p.minTaskExecutionTime == 0 || executionTime < p.minTaskExecutionTime {
				p.minTaskExecutionTime = executionTime
			}
			p.mu.Unlock()

			// 检查是否有 panic，在锁外处理，避免死锁
			if panicErr != nil {
				log.Printf("[ants][%s] 捕获到panic: %v", p.name, panicErr)
				atomic.AddInt64(&p.failedTasks, 1)
				// 调用错误回调函数
				p.callErrorCallbacks(&PoolError{
					Code:    ErrCodeTaskPanic,
					Message: "任务执行过程中发生panic",
					Err:     fmt.Errorf("%v", panicErr),
				})
			} else if ctxCanceled {
				// 上下文被取消，记录为失败任务
				atomic.AddInt64(&p.failedTasks, 1)
				// 调用错误回调函数
				p.callErrorCallbacks(&PoolError{
					Code:    ErrCodeContextCanceled,
					Message: "上下文已取消",
					Err:     ctx.Err(),
				})
			}
		}()

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			// 上下文被取消
			ctxCanceled = true
			return
		default:
			// 执行任务
			task(ctx)
		}
	}

	err := p.pool.Submit(wrappedTask)
	if err != nil {
		atomic.AddInt64(&p.failedTasks, 1)
		poolErr := &PoolError{
			Code:    ErrCodeSubmitTask,
			Message: "提交任务失败",
			Err:     err,
		}
		// 调用错误回调函数
		p.callErrorCallbacks(poolErr)
		return poolErr
	}

	return nil
}

// SubmitWithTimeout 提交带有超时的任务
func (p *Pool) SubmitWithTimeout(timeout time.Duration, task func()) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return p.SubmitWithContext(ctx, func(ctx context.Context) {
		task()
	})
}

// Tune 动态调整线程池大小
func (p *Pool) Tune(size int) {
	p.pool.Tune(size)
}

// Running 返回当前运行的协程数
func (p *Pool) Running() int {
	return p.pool.Running()
}

// Waiting 返回等待中的任务数
func (p *Pool) Waiting() int {
	return p.pool.Waiting()
}

// Metrics 返回当前线程池的监控指标
func (p *Pool) Metrics() Metrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 计算平均任务执行时间
	var avgExecutionTime time.Duration
	if len(p.taskExecutionTimes) > 0 {
		var total time.Duration
		for _, t := range p.taskExecutionTimes {
			total += t
		}
		avgExecutionTime = total / time.Duration(len(p.taskExecutionTimes))
	}

	// 获取当前队列长度
	currentQueueLength := p.pool.Waiting()

	// 计算队列长度变化
	queueLengthChange := currentQueueLength - p.lastQueueLength

	// 更新上一次队列长度
	// 注意：这里需要写锁，所以在 RLock 之外更新
	p.mu.RUnlock()
	p.mu.Lock()
	p.lastQueueLength = currentQueueLength

	// 复制健康分析数据（避免并发问题）
	slowTasksCopy := make([]SlowTaskRecord, len(p.slowTaskRecords))
	copy(slowTasksCopy, p.slowTaskRecords)

	failedTasksCopy := make([]FailedTaskRecord, len(p.failedTaskRecords))
	copy(failedTasksCopy, p.failedTaskRecords)

	p.mu.Unlock()
	p.mu.RLock()

	// 计算成功任务数
	successTasks := atomic.LoadInt64(&p.totalTasks) - atomic.LoadInt64(&p.failedTasks)

	return Metrics{
		Name:              p.name,
		RunningGoroutines: p.pool.Running(),
		WaitingTasks:      currentQueueLength,
		TotalTasks:        atomic.LoadInt64(&p.totalTasks),
		FailedTasks:       atomic.LoadInt64(&p.failedTasks),
		PoolCapacity:      p.pool.Cap(),
		PoolSize:          p.pool.Running(),
		Timestamp:         time.Now(),
		// 新增指标
		TaskExecutionTime:    avgExecutionTime,
		MaxTaskExecutionTime: p.maxTaskExecutionTime,
		MinTaskExecutionTime: p.minTaskExecutionTime,
		QueueLengthChange:    queueLengthChange,
		GoroutinesCreated:    p.goroutinesCreated,
		GoroutinesDestroyed:  p.goroutinesDestroyed,
		SuccessTasks:         successTasks,
		// 健康分析指标
		TrackedTasks:    atomic.LoadInt64(&p.trackedTasks),
		SlowTasks:       slowTasksCopy,
		FailedTasksList: failedTasksCopy,
	}
}

// Release 释放线程池资源
func (p *Pool) Release() error {
	return p.pool.ReleaseTimeout(time.Second * 5)
}

// RegisterErrorCallback 注册错误回调函数，用于处理池中的错误
func (p *Pool) RegisterErrorCallback(callback ErrorCallback) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorCallbacks = append(p.errorCallbacks, callback)
}

// 调用所有错误回调函数
func (p *Pool) callErrorCallbacks(err *PoolError) {
	p.mu.RLock()
	callbacks := make([]ErrorCallback, len(p.errorCallbacks))
	copy(callbacks, p.errorCallbacks)
	p.mu.RUnlock()

	for _, callback := range callbacks {
		callback(err)
	}
}

// PoolWithFunc 是对 ants.PoolWithFunc 的封装
type PoolWithFunc struct {
	pool           *ants.PoolWithFunc
	name           string
	totalTasks     int64
	failedTasks    int64
	errorCallbacks []ErrorCallback // 错误回调函数列表
	// 新增字段用于跟踪监控指标
	taskExecutionTimes   []time.Duration // 任务执行时间切片
	maxTaskExecutionTime time.Duration   // 最大任务执行时间
	minTaskExecutionTime time.Duration   // 最小任务执行时间
	goroutinesCreated    int64           // 已创建的协程总数
	goroutinesDestroyed  int64           // 已销毁的协程总数
	lastQueueLength      int             // 上一次的队列长度
	mu                   sync.RWMutex    // 读写锁，保护共享字段
	// 健康分析相关字段
	healthConfig      HealthConfig       // 健康分析配置
	trackedTasks      int64              // 被追踪的任务总数
	slowTaskRecords   []SlowTaskRecord   // 慢任务记录（环形缓冲）
	failedTaskRecords []FailedTaskRecord // 异常任务记录（环形缓冲）
}

// NewPoolWithFunc 创建一个新的带函数的线程池
func NewPoolWithFunc(name string, workerFunc func(interface{}), size ...int) (*PoolWithFunc, error) {
	if len(size) > 0 {
		return NewPoolWithFuncWithOptions(name, workerFunc, size[0])
	}
	return NewPoolWithFuncWithOptions(name, workerFunc, 0)
}

// NewPoolWithFuncWithOptions 创建一个新的带函数的线程池，支持自定义配置选项
func NewPoolWithFuncWithOptions(name string, workerFunc func(interface{}), size int, optionFuncs ...func(*ants.Options)) (*PoolWithFunc, error) {
	poolSize := runtime.NumCPU() * 2
	if size > 0 {
		poolSize = size
	}

	// 先创建池实例
	pool := &PoolWithFunc{
		name:              name,
		healthConfig:      DefaultHealthConfig(), // 使用默认健康分析配置
		slowTaskRecords:   make([]SlowTaskRecord, 0),
		failedTaskRecords: make([]FailedTaskRecord, 0),
	}

	wrappedFunc := func(arg interface{}) {
		startTime := time.Now()
		var panicErr interface{}

		// 检查是否为带ID的任务
		var taskID string
		var actualArg interface{}
		var isTracked bool

		if wrappedTask, ok := arg.(taskWithID); ok {
			taskID = wrappedTask.TaskID
			actualArg = wrappedTask.Arg
			isTracked = true
		} else {
			actualArg = arg
			isTracked = false
		}

		defer func() {
			panicErr = recover()
			// 计算任务执行时间
			executionTime := time.Since(startTime)

			// 更新监控指标
			pool.mu.Lock()

			// 记录任务执行时间
			pool.taskExecutionTimes = append(pool.taskExecutionTimes, executionTime)
			// 限制切片长度，避免内存占用过高
			if len(pool.taskExecutionTimes) > 1000 {
				pool.taskExecutionTimes = pool.taskExecutionTimes[len(pool.taskExecutionTimes)-1000:]
			}

			// 更新最大和最小执行时间
			if executionTime > pool.maxTaskExecutionTime {
				pool.maxTaskExecutionTime = executionTime
			}
			if pool.minTaskExecutionTime == 0 || executionTime < pool.minTaskExecutionTime {
				pool.minTaskExecutionTime = executionTime
			}

			// 如果是被追踪的任务且启用了健康分析
			if isTracked && pool.healthConfig.Enabled {
				// 检查是否为慢任务
				if pool.healthConfig.SlowTaskThreshold > 0 && executionTime > pool.healthConfig.SlowTaskThreshold {
					record := SlowTaskRecord{
						TaskID:        taskID,
						ExecutionTime: executionTime,
						Timestamp:     time.Now(),
					}
					pool.slowTaskRecords = append(pool.slowTaskRecords, record)

					// 限制记录数量
					maxRecords := pool.healthConfig.MaxSlowTaskRecords
					if len(pool.slowTaskRecords) > maxRecords {
						pool.slowTaskRecords = pool.slowTaskRecords[len(pool.slowTaskRecords)-maxRecords:]
					}
				}

				// 记录异常任务
				if panicErr != nil {
					record := FailedTaskRecord{
						TaskID:    taskID,
						Error:     fmt.Sprintf("panic: %v", panicErr),
						Timestamp: time.Now(),
					}
					pool.failedTaskRecords = append(pool.failedTaskRecords, record)

					// 限制记录数量
					maxRecords := pool.healthConfig.MaxFailedTaskRecords
					if len(pool.failedTaskRecords) > maxRecords {
						pool.failedTaskRecords = pool.failedTaskRecords[len(pool.failedTaskRecords)-maxRecords:]
					}
				}
			}

			pool.mu.Unlock()

			// 检查是否有 panic
			if panicErr != nil {
				if isTracked {
					log.Printf("[ants][%s] 任务 %s 捕获到panic: %v", name, taskID, panicErr)
				} else {
					log.Printf("[ants][%s] 捕获到panic: %v", name, panicErr)
				}
				atomic.AddInt64(&pool.failedTasks, 1)
				// 调用错误回调函数
				pool.callErrorCallbacks(&PoolError{
					Code:    ErrCodeTaskPanic,
					Message: "任务执行过程中发生panic",
					Err:     fmt.Errorf("%v", panicErr),
				})
			}
		}()
		workerFunc(actualArg)
	}

	// 默认配置
	defaultOptions := []ants.Option{
		ants.WithExpiryDuration(time.Minute),
		ants.WithMaxBlockingTasks(poolSize * 3), // 默认任务队列大小设为池大小的3倍
	}

	// 应用默认选项
	options := ants.Options{}
	for _, opt := range defaultOptions {
		opt(&options)
	}

	// 应用用户自定义选项
	for _, optFunc := range optionFuncs {
		optFunc(&options)
	}

	// 转换为 ants.Option 切片
	optionSlice := []ants.Option{}
	if options.ExpiryDuration > 0 {
		optionSlice = append(optionSlice, ants.WithExpiryDuration(options.ExpiryDuration))
	}
	if options.MaxBlockingTasks > 0 {
		optionSlice = append(optionSlice, ants.WithMaxBlockingTasks(options.MaxBlockingTasks))
	}
	if options.PreAlloc {
		optionSlice = append(optionSlice, ants.WithPreAlloc(options.PreAlloc))
	}
	if options.Nonblocking {
		optionSlice = append(optionSlice, ants.WithNonblocking(options.Nonblocking))
	}
	if options.Logger != nil {
		optionSlice = append(optionSlice, ants.WithLogger(options.Logger))
	}

	antsPool, err := ants.NewPoolWithFunc(poolSize, wrappedFunc, optionSlice...)
	if err != nil {
		poolErr := &PoolError{
			Code:    ErrCodeCreatePool,
			Message: "创建带函数的线程池失败",
			Err:     err,
		}
		return nil, poolErr
	}

	pool.pool = antsPool
	// 初始化时记录创建的协程数
	atomic.AddInt64(&pool.goroutinesCreated, int64(poolSize))

	return pool, nil
}

// SetHealthConfig 设置健康分析配置
func (p *PoolWithFunc) SetHealthConfig(config HealthConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthConfig = config
}

// GetHealthConfig 获取健康分析配置
func (p *PoolWithFunc) GetHealthConfig() HealthConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.healthConfig
}

// taskWithID 带任务ID的任务参数
type taskWithID struct {
	TaskID string
	Arg    interface{}
}

// InvokeWithTaskID 提交带任务ID的任务到 PoolWithFunc（会被健康分析追踪）
func (p *PoolWithFunc) InvokeWithTaskID(taskID string, arg interface{}) error {
	atomic.AddInt64(&p.totalTasks, 1)

	// 检查是否启用健康分析
	p.mu.RLock()
	healthEnabled := p.healthConfig.Enabled
	p.mu.RUnlock()

	if healthEnabled {
		atomic.AddInt64(&p.trackedTasks, 1)
	}

	// 包装参数，传递任务ID
	wrappedArg := taskWithID{
		TaskID: taskID,
		Arg:    arg,
	}

	err := p.pool.Invoke(wrappedArg)
	if err != nil {
		atomic.AddInt64(&p.failedTasks, 1)
		poolErr := &PoolError{
			Code:    ErrCodeSubmitTask,
			Message: fmt.Sprintf("提交任务 %s 到 PoolWithFunc 失败", taskID),
			Err:     err,
		}
		// 调用错误回调函数
		p.callErrorCallbacks(poolErr)
		return poolErr
	}

	return nil
}

// Invoke 提交任务到 PoolWithFunc
func (p *PoolWithFunc) Invoke(arg interface{}) error {
	atomic.AddInt64(&p.totalTasks, 1)

	err := p.pool.Invoke(arg)
	if err != nil {
		atomic.AddInt64(&p.failedTasks, 1)
		poolErr := &PoolError{
			Code:    ErrCodeSubmitTask,
			Message: "提交任务到 PoolWithFunc 失败",
			Err:     err,
		}
		// 调用错误回调函数
		p.callErrorCallbacks(poolErr)
		return poolErr
	}

	return nil
}

// Tune 动态调整线程池大小
func (p *PoolWithFunc) Tune(size int) {
	p.pool.Tune(size)
}

// Running 返回当前运行的协程数
func (p *PoolWithFunc) Running() int {
	return p.pool.Running()
}

// Waiting 返回等待中的任务数
func (p *PoolWithFunc) Waiting() int {
	return p.pool.Waiting()
}

// Metrics 返回当前线程池的监控指标
func (p *PoolWithFunc) Metrics() Metrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 计算平均任务执行时间
	var avgExecutionTime time.Duration
	if len(p.taskExecutionTimes) > 0 {
		var total time.Duration
		for _, t := range p.taskExecutionTimes {
			total += t
		}
		avgExecutionTime = total / time.Duration(len(p.taskExecutionTimes))
	}

	// 获取当前队列长度
	currentQueueLength := p.pool.Waiting()

	// 计算队列长度变化
	queueLengthChange := currentQueueLength - p.lastQueueLength

	// 更新上一次队列长度
	// 注意：这里需要写锁，所以在 RLock 之外更新
	p.mu.RUnlock()
	p.mu.Lock()
	p.lastQueueLength = currentQueueLength

	// 复制健康分析数据（避免并发问题）
	slowTasksCopy := make([]SlowTaskRecord, len(p.slowTaskRecords))
	copy(slowTasksCopy, p.slowTaskRecords)

	failedTasksCopy := make([]FailedTaskRecord, len(p.failedTaskRecords))
	copy(failedTasksCopy, p.failedTaskRecords)

	p.mu.Unlock()
	p.mu.RLock()

	// 计算成功任务数
	successTasks := atomic.LoadInt64(&p.totalTasks) - atomic.LoadInt64(&p.failedTasks)

	return Metrics{
		Name:              p.name,
		RunningGoroutines: p.pool.Running(),
		WaitingTasks:      currentQueueLength,
		TotalTasks:        atomic.LoadInt64(&p.totalTasks),
		FailedTasks:       atomic.LoadInt64(&p.failedTasks),
		PoolCapacity:      p.pool.Cap(),
		PoolSize:          p.pool.Running(),
		Timestamp:         time.Now(),
		// 新增指标
		TaskExecutionTime:    avgExecutionTime,
		MaxTaskExecutionTime: p.maxTaskExecutionTime,
		MinTaskExecutionTime: p.minTaskExecutionTime,
		QueueLengthChange:    queueLengthChange,
		GoroutinesCreated:    p.goroutinesCreated,
		GoroutinesDestroyed:  p.goroutinesDestroyed,
		SuccessTasks:         successTasks,
		// 健康分析指标
		TrackedTasks:    atomic.LoadInt64(&p.trackedTasks),
		SlowTasks:       slowTasksCopy,
		FailedTasksList: failedTasksCopy,
	}
}

// Release 释放线程池资源
func (p *PoolWithFunc) Release() error {
	return p.pool.ReleaseTimeout(time.Second * 5)
}

// RegisterErrorCallback 注册错误回调函数，用于处理池中的错误
func (p *PoolWithFunc) RegisterErrorCallback(callback ErrorCallback) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorCallbacks = append(p.errorCallbacks, callback)
}

// 调用所有错误回调函数
func (p *PoolWithFunc) callErrorCallbacks(err *PoolError) {
	p.mu.RLock()
	callbacks := make([]ErrorCallback, len(p.errorCallbacks))
	copy(callbacks, p.errorCallbacks)
	p.mu.RUnlock()

	for _, callback := range callbacks {
		callback(err)
	}
}
