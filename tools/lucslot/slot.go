package lucslot

/**
  插槽消息： 主要用于精准执行 延迟任务的 圆形插槽
  作者：     卢闯  2019-08-29
  描述： 查找每秒转动一格，会把格子中的任务取出来执行
         存放任务进格子非常重要：要自动算出格子正确位置
  项目细节：格子中的任务太多，导致一秒之内无法完成任务分配的问题已经解决，格子不会无故跳过，否则会导致延迟1小时
  后期优化： 支持任务缓存在硬盘或内存，一定程度保重任务不会丢失
             思路从硬盘或内存读取加载：删除或执行已经过时任务，重新开始转动
*/
import (
	"fmt"
	"sync"
	"time"
)

//圆形插槽实体
type ArchetypeSlots struct {
	//环形插槽共3600个，每个插槽可放N个待执行任务
	slots Slot
	//当前插槽
	curIndex    int
	stop        chan bool
	globAllKeys sync.Map
	checkTime   time.Time // 校准时间
	Now         time.Time // 槽对应的时间
	//firstRunTime time.Time
}

func newSlots() *ArchetypeSlots {
	as := &ArchetypeSlots{
		curIndex: 0,
		stop:     make(chan bool),
	}
	//初始化每个插槽： 任务用map存放【每个任务名称不能相同，防止任务被重复执行】
	for i := 0; i < 3600; i++ {
		as.slots.m[i] = make(map[string]*Task)
	}
	// 去掉时区的影响
	as.Now, _ = time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02 15:04:05"))
	as.checkTime, _ = time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02 15")+":00:00")

	//as.globAllKeys = make(map[string]string)
	return as
}

//开始运行
func (as *ArchetypeSlots) Run() {
	defer func() {
		fmt.Println("luc slots exit")
	}()
	now, _ := time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02 15:04:05"))
	// 时间+1，否则索引会慢一秒
	subSecond := now.Unix() - as.checkTime.Unix() + 1
	as.curIndex = int(subSecond % 3600)
	as.Now = now
	tick := time.NewTicker(time.Second) //一秒执行一次
	for {
		select {
		case <-as.stop:
			return
		case <-tick.C:
			{
				//wg.Add(1)
				as.Now = as.Now.Add(time.Second)
				as.taskLoop()
				if as.curIndex == 3599 {
					as.curIndex = 0
					as.checkTime = as.checkTime.Add(time.Hour)
				} else {
					as.curIndex++
				}
				//wg.Wait()
			}
		}
	}
}

//停止运行
func (as *ArchetypeSlots) Stop() {
	as.stop <- true
}
