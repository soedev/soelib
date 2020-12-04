package lucslot

import (
	"fmt"
	"github.com/soedev/soelib/tools/ants"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

//1000000 个任务调度测试
func TestSlots(t *testing.T) {
	var wg sync.WaitGroup
	//先初始化池子
	ants.InitCoroutinePool()
	dm := newSlots()
	ants.SubmitTask(func() {
		dm.Run()
	})
	for i := 0; i < 100; i++ {
		wg.Add(1)
		s := rand.Intn(2000) + 3600
		task := strconv.Itoa(i)
		runTime := time.Now().Add(time.Second * time.Duration(s))
		_ = dm.AddTask(runTime, task, func(args ...interface{}) {
			//模拟延迟
			fmt.Println("计划运行时间:" + args[0].(string) + ",当前时间:" + time.Now().Format("15:04:05"))
			d := rand.Intn(15)
			time.Sleep(time.Second * time.Duration(d))
			//fmt.Println(fmt.Sprintf("%d 秒后 任务：%s 花费时间：%d s", s, task, d))
			wg.Done()
		}, []interface{}{runTime.Format("15:04:05")}, true)
	}
	wg.Wait()
	fmt.Println("end")
}

func TestSlots1(t *testing.T) {
	//先初始化池子
	ants.InitCoroutinePool()
	//创建延迟消息
	dm := newSlots()
	//添加任务 五秒后执行的任务
	_ = dm.AddTask(time.Now().Add(time.Second*5), "test1", task, []interface{}{1}, false)

	_ = dm.AddTask(time.Now().Add(time.Second*10), "test2", func(args ...interface{}) {
		fmt.Println(args...)
	}, []interface{}{1, 2, 3}, false)

	_ = dm.AddTask(time.Now().Add(time.Second*10), "test3", func(args ...interface{}) {
		fmt.Println(args...)
	}, []interface{}{4, 5, 6}, true)

	err := dm.AddTask(time.Now().Add(time.Second*10), "test3", func(args ...interface{}) {
		fmt.Println(args...)
	}, []interface{}{4, 5, 6}, true)

	if err != nil {
		fmt.Println(err.Error())
	}

	//40秒后关闭
	time.AfterFunc(time.Second*30, func() {
		dm.Stop()
		fmt.Println(fmt.Sprintf("Running %d", ants.RunningCount()))
	})
	dm.Run()
}

func task(args ...interface{}) {
	fmt.Println(args[0])
}
