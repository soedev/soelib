package lucslot

import (
	"github.com/soedev/soelib/tools/ants"
	"time"
)

var archetypeSlots *ArchetypeSlots

func InitDelayTaskQuery() {
	archetypeSlots = newSlots()
	ants.SubmitTask(func() {
		archetypeSlots.Run()
	})

}

//AddDelayTask 添加自定执行任务 t 执行时间  taskName 任务名称 exec 执行方法 params 执行参数
func AddDelayTask(t time.Time, taskName string, exec TaskFunc, params []interface{}, unique bool) error {
	return archetypeSlots.AddTask(t, taskName, exec, params, unique)
}

//Release 停止运行
func Release() {
	archetypeSlots.Stop()
}
