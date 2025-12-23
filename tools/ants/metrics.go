package ants

import (
	"runtime"
	"sync/atomic"
	"time"
)

// SlowTaskRecord 慢任务记录
type SlowTaskRecord struct {
	TaskID        string        `json:"task_id"`        // 任务ID
	ExecutionTime time.Duration `json:"execution_time"` // 执行时间
	Timestamp     time.Time     `json:"timestamp"`      // 记录时间
}

// FailedTaskRecord 异常任务记录
type FailedTaskRecord struct {
	TaskID    string    `json:"task_id"`   // 任务ID
	Error     string    `json:"error"`     // 错误信息
	Timestamp time.Time `json:"timestamp"` // 记录时间
}

// Metrics 包含线程池的所有监控指标
type Metrics struct {
	Name              string    `json:"name"`
	RunningGoroutines int       `json:"running_goroutines"` // 运行中的协程数
	WaitingTasks      int       `json:"waiting_tasks"`      // 等待中的任务数
	TotalTasks        int64     `json:"total_tasks"`        // 总任务数
	FailedTasks       int64     `json:"failed_tasks"`       // 失败任务数
	PoolCapacity      int       `json:"pool_capacity"`      // 池容量
	PoolSize          int       `json:"pool_size"`          // 当前池大小
	Timestamp         time.Time `json:"timestamp"`          // 指标生成时间
	// 新增指标
	TaskExecutionTime    time.Duration `json:"task_execution_time"`     // 任务执行时间（平均值，纳秒）
	MaxTaskExecutionTime time.Duration `json:"max_task_execution_time"` // 最大任务执行时间（纳秒）
	MinTaskExecutionTime time.Duration `json:"min_task_execution_time"` // 最小任务执行时间（纳秒）
	QueueLengthChange    int           `json:"queue_length_change"`     // 队列长度变化
	GoroutinesCreated    int64         `json:"goroutines_created"`      // 已创建的协程总数
	GoroutinesDestroyed  int64         `json:"goroutines_destroyed"`    // 已销毁的协程总数
	SuccessTasks         int64         `json:"success_tasks"`           // 成功任务数
	// 健康分析指标（仅针对带ID的任务）
	TrackedTasks    int64              `json:"tracked_tasks"`     // 被追踪的任务总数
	SlowTasks       []SlowTaskRecord   `json:"slow_tasks"`        // 慢任务列表
	FailedTasksList []FailedTaskRecord `json:"failed_tasks_list"` // 异常任务列表
}

// NewMetrics 创建一个新的 Metrics 实例
func NewMetrics(name string) Metrics {
	return Metrics{
		Name:      name,
		Timestamp: time.Now(),
	}
}

// Update 使用当前线程池状态更新指标
func (m *Metrics) Update(pool interface{}) {
	m.Timestamp = time.Now()

	switch p := pool.(type) {
	case *Pool:
		m.RunningGoroutines = p.Running()
		m.WaitingTasks = p.Waiting()
		m.TotalTasks = atomic.LoadInt64(&p.totalTasks)
		m.FailedTasks = atomic.LoadInt64(&p.failedTasks)
		m.PoolCapacity = p.pool.Cap() // 添加池容量指标
		// 更新成功任务数
		m.SuccessTasks = m.TotalTasks - m.FailedTasks
		// 更新任务执行时间指标
		p.mu.RLock()
		if len(p.taskExecutionTimes) > 0 {
			var total time.Duration
			for _, t := range p.taskExecutionTimes {
				total += t
			}
			m.TaskExecutionTime = total / time.Duration(len(p.taskExecutionTimes))
			m.MaxTaskExecutionTime = p.maxTaskExecutionTime
			m.MinTaskExecutionTime = p.minTaskExecutionTime
		}
		m.GoroutinesCreated = p.goroutinesCreated
		m.GoroutinesDestroyed = p.goroutinesDestroyed
		p.mu.RUnlock()
	case *PoolWithFunc:
		m.RunningGoroutines = p.Running()
		m.WaitingTasks = p.Waiting()
		m.TotalTasks = atomic.LoadInt64(&p.totalTasks)
		m.FailedTasks = atomic.LoadInt64(&p.failedTasks)
		m.PoolCapacity = p.pool.Cap() // 添加池容量指标
		// 更新成功任务数
		m.SuccessTasks = m.TotalTasks - m.FailedTasks
		// 更新任务执行时间指标
		p.mu.RLock()
		if len(p.taskExecutionTimes) > 0 {
			var total time.Duration
			for _, t := range p.taskExecutionTimes {
				total += t
			}
			m.TaskExecutionTime = total / time.Duration(len(p.taskExecutionTimes))
			m.MaxTaskExecutionTime = p.maxTaskExecutionTime
			m.MinTaskExecutionTime = p.minTaskExecutionTime
		}
		m.GoroutinesCreated = p.goroutinesCreated
		m.GoroutinesDestroyed = p.goroutinesDestroyed
		p.mu.RUnlock()
	}
}

// Config 线程池的配置结构
type Config struct {
	Size             int           `json:"size"`               // 线程池大小
	ExpiryDuration   time.Duration `json:"expiry_duration"`    // 工作协程过期时间
	PreAlloc         bool          `json:"pre_alloc"`          // 是否预分配工作协程
	MaxBlockingTasks int           `json:"max_blocking_tasks"` // 最大阻塞任务数
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Size:             runtime.NumCPU() * 2,
		ExpiryDuration:   time.Minute,
		PreAlloc:         false,
		MaxBlockingTasks: 0,
	}
}
