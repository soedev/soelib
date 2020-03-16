package task

import (
	"fmt"
	"sync"
	"time"

	"github.com/alex023/clock"
)

//TimeTaskService 任务服务
type TimeTaskService struct {
	sync.Mutex
	cache map[string]taskjob
	clock *clock.Clock
}

//TimeTaskDTO 任务DTO
type TimeTaskDTO struct {
	Name    string
	Timeout time.Duration
	JobFunc func()
}

type taskjob struct {
	name string
	job  clock.Job
}

const (
	taskTime   = 1 //按次数执行
	taskRepeat = 2 //重复执行
)

var taskServiceIns *TimeTaskService
var once sync.Once

//GetTimeTaskIns 单例话任务服务
func GetTimeTaskIns() *TimeTaskService {
	once.Do(func() {
		taskServiceIns = &TimeTaskService{
			cache: make(map[string]taskjob),
			clock: clock.NewClock(),
		}
	})
	return taskServiceIns
}

//AddTask 新增任务
func (s *TimeTaskService) AddTask(taskDTO TimeTaskDTO) {
	s.Lock()
	defer s.Unlock()

	_, founded := s.cache[taskDTO.Name]
	if founded {
		s.RemoveTask(taskDTO.Name)
	}

	job, _ := s.clock.AddJobRepeat(taskDTO.Timeout, 0, taskDTO.JobFunc)
	item := taskjob{
		name: taskDTO.Name,
		job:  job,
	}
	s.cache[taskDTO.Name] = item

	fmt.Printf("%v| [ added ] taskName [%v] duration=[%2d] \n", time.Now().Format("15:04:05"), taskDTO.Name, int(taskDTO.Timeout.Seconds()))

}

//AddTaskWithInterval 新增定时任务
func (s *TimeTaskService) AddTaskWithInterval(taskDTO TimeTaskDTO) {
	s.Lock()
	defer s.Unlock()

	item1, founded := s.cache[taskDTO.Name]
	if founded {
		delete(s.cache, taskDTO.Name)
		item1.job.Cancel()
	}

	job, _ := s.clock.AddJobWithInterval(taskDTO.Timeout, taskDTO.JobFunc)
	item := taskjob{
		name: taskDTO.Name,
		job:  job,
	}
	s.cache[taskDTO.Name] = item
	fmt.Printf("%v| [ added ] taskName [%v] duration=[%2d] \n", time.Now().Format("15:04:05"), taskDTO.Name, int(taskDTO.Timeout.Seconds()))

}

//CheckTask 校验任务是否存在
func (s *TimeTaskService) CheckTask(taskName string) bool {
	s.Lock()
	defer s.Unlock()
	_, founded := s.cache[taskName]
	return founded
}

//GetTaskNum 获取任务数量
func (s *TimeTaskService) GetTaskNum() int {
	s.Lock()
	defer s.Unlock()

	return len(s.cache)
}

//RemoveTask 移除任务
func (s *TimeTaskService) RemoveTask(taskName string) {
	s.Lock()
	defer s.Unlock()
	fmt.Printf("%v| [removed] token [%v] by Task \n", time.Now().Format("15:04:05"), taskName)
	if taskjob, founded := s.cache[taskName]; founded {
		delete(s.cache, taskName)
		taskjob.job.Cancel()
	}
}

//RemoveAll 移除任务
func (s *TimeTaskService) RemoveAll() {
	s.Lock()
	defer s.Unlock()
	for taskName, taskjob := range s.cache {
		delete(s.cache, taskName)
		taskjob.job.Cancel()
	}
}
