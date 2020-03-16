package task

import (
	"fmt"
	"testing"
	"time"
)

func TestTimeTaskService_AddTask(t *testing.T) {
	fn := func() {

		fmt.Println("schedule1")
	}
	GetTimeTaskIns().AddTask(TimeTaskDTO{
		Name:    "task_1",
		Timeout: time.Millisecond * 200,
		JobFunc: fn,
	})

	fn1 := func() {
		time.Sleep(time.Millisecond * 300)
		fmt.Println("schedule2")
	}

	GetTimeTaskIns().AddTask(TimeTaskDTO{
		Name:    "task_2",
		Timeout: time.Millisecond * 200,
		JobFunc: fn1,
	})

	time.Sleep(time.Second * 1)
}

func TestTimeTaskService_AddTaskWithInterval(t *testing.T) {
	fn := func() {

		fmt.Println(time.Now().Format("15:04:05") + "  schedule1")
	}
	GetTimeTaskIns().AddTaskWithInterval(TimeTaskDTO{
		Name:    "task_1",
		Timeout: time.Second * 3,
		JobFunc: fn,
	})

	time.Sleep(time.Second * 3)

	GetTimeTaskIns().AddTaskWithInterval(TimeTaskDTO{
		Name:    "task_1",
		Timeout: time.Second * 1,
		JobFunc: fn,
	})

	time.Sleep(time.Second * 3)

}
