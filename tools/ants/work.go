package ants

var antsPool *Pool

//初始化协程池：默认池子大小 很大很大
func InitCoroutinePool() {
	antsPool, _ = NewPool(DEFAULT_ANTS_POOL_SIZE)

}
func SubmitTask(task func()) {
	_ = antsPool.Submit(task)
}
func CoroutineRelease() {
	antsPool.Release()
}

func RunningCount() int {
	return antsPool.Running()
}
