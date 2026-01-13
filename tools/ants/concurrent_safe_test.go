package ants

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// TestSubmitTaskWithData 测试 SubmitTaskWithData 方法
func TestSubmitTaskWithData(t *testing.T) {
	// 初始化全局池
	InitCoroutinePool()
	defer CoroutineRelease()

	// 测试传递 map
	t.Run("传递map数据", func(t *testing.T) {
		data := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": map[string]string{
				"nested": "value",
			},
		}

		var wg sync.WaitGroup
		wg.Add(1)

		err := SubmitTaskWithData(data, func(d interface{}) {
			defer wg.Done()

			// 验证数据
			dataMap, ok := d.(map[string]interface{})
			if !ok {
				t.Error("数据类型转换失败")
				return
			}

			if dataMap["key1"] != "value1" {
				t.Errorf("期望 key1=value1, 得到 %v", dataMap["key1"])
			}

			// 验证可以安全序列化
			_, err := json.Marshal(dataMap)
			if err != nil {
				t.Errorf("序列化失败: %v", err)
			}
		})

		if err != nil {
			t.Errorf("提交任务失败: %v", err)
		}

		wg.Wait()
	})

	// 测试传递 struct（注意：JSON 反序列化会转为 map）
	t.Run("传递struct数据", func(t *testing.T) {
		type TestStruct struct {
			ID   int
			Name string
		}

		data := TestStruct{
			ID:   123,
			Name: "test",
		}

		var wg sync.WaitGroup
		wg.Add(1)

		err := SubmitTaskWithData(data, func(d interface{}) {
			defer wg.Done()

			// JSON 反序列化 struct 会转换为 map[string]interface{}
			dataMap, ok := d.(map[string]interface{})
			if !ok {
				t.Error("数据类型转换失败")
				return
			}

			// 验证数据（注意：数字会转为 float64）
			if dataMap["Name"] != "test" {
				t.Errorf("Name 不匹配: %v", dataMap["Name"])
			}

			id, ok := dataMap["ID"].(float64)
			if !ok || int(id) != 123 {
				t.Errorf("ID 不匹配: %v", dataMap["ID"])
			}
		})

		if err != nil {
			t.Errorf("提交任务失败: %v", err)
		}

		wg.Wait()
	})
}

// TestSubmitTaskGeneric 测试泛型方法
func TestSubmitTaskGeneric(t *testing.T) {
	InitCoroutinePool()
	defer CoroutineRelease()

	// 测试传递 struct
	t.Run("泛型传递struct", func(t *testing.T) {
		type User struct {
			ID    int
			Name  string
			Email string
		}

		user := User{
			ID:    123,
			Name:  "test",
			Email: "test@example.com",
		}

		var wg sync.WaitGroup
		wg.Add(1)

		err := SubmitTaskGeneric(user, func(u User) {
			defer wg.Done()

			// 类型安全，无需类型断言
			if u.ID != 123 {
				t.Errorf("期望 ID=123, 得到 %d", u.ID)
			}
			if u.Name != "test" {
				t.Errorf("期望 Name=test, 得到 %s", u.Name)
			}
		})

		if err != nil {
			t.Errorf("提交任务失败: %v", err)
		}

		wg.Wait()
	})

	// 测试传递 map
	t.Run("泛型传递map", func(t *testing.T) {
		data := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		var wg sync.WaitGroup
		wg.Add(1)

		err := SubmitTaskGeneric(data, func(d map[string]string) {
			defer wg.Done()

			// 类型安全
			if d["key1"] != "value1" {
				t.Errorf("期望 key1=value1, 得到 %s", d["key1"])
			}
		})

		if err != nil {
			t.Errorf("提交任务失败: %v", err)
		}

		wg.Wait()
	})

	// 测试传递 slice
	t.Run("泛型传递slice", func(t *testing.T) {
		data := []int{1, 2, 3, 4, 5}

		var wg sync.WaitGroup
		wg.Add(1)

		err := SubmitTaskGeneric(data, func(d []int) {
			defer wg.Done()

			if len(d) != 5 {
				t.Errorf("期望长度=5, 得到 %d", len(d))
			}
			if d[0] != 1 || d[4] != 5 {
				t.Errorf("数据不匹配: %v", d)
			}
		})

		if err != nil {
			t.Errorf("提交任务失败: %v", err)
		}

		wg.Wait()
	})
}

// TestConcurrentSafety 测试并发安全性
func TestConcurrentSafety(t *testing.T) {
	InitCoroutinePool()
	defer CoroutineRelease()

	t.Run("并发访问map", func(t *testing.T) {
		// 创建一个 map，模拟可能被并发访问的数据
		data := map[string]int{
			"count": 0,
		}

		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		// 提交100个任务，每个任务都使用独立的数据副本
		for i := 0; i < 100; i++ {
			wg.Add(1)
			data["count"] = i

			err := SubmitTaskGeneric(data, func(d map[string]int) {
				defer wg.Done()

				// 每个任务都应该能安全地序列化数据
				_, err := json.Marshal(d)
				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			})

			if err != nil {
				t.Errorf("提交任务失败: %v", err)
			}
		}

		wg.Wait()

		if successCount != 100 {
			t.Errorf("期望成功100次, 实际成功 %d 次", successCount)
		}
	})
}

// TestDataIsolation 测试数据隔离
func TestDataIsolation(t *testing.T) {
	InitCoroutinePool()
	defer CoroutineRelease()

	t.Run("数据隔离验证", func(t *testing.T) {
		type Task struct {
			ID   int
			Data []int
		}

		// 原始数据
		original := Task{
			ID:   1,
			Data: []int{1, 2, 3},
		}

		var wg sync.WaitGroup
		wg.Add(1)

		err := SubmitTaskGeneric(original, func(t Task) {
			defer wg.Done()

			// 修改副本
			t.ID = 999
			t.Data[0] = 999
		})

		if err != nil {
			t.Errorf("提交任务失败: %v", err)
		}

		wg.Wait()

		// 验证原始数据未被修改
		if original.ID != 1 {
			t.Errorf("原始数据被修改: ID=%d", original.ID)
		}
		if original.Data[0] != 1 {
			t.Errorf("原始数据被修改: Data[0]=%d", original.Data[0])
		}
	})
}

// BenchmarkSubmitTask 基准测试：普通提交
func BenchmarkSubmitTask(b *testing.B) {
	InitCoroutinePool()
	defer CoroutineRelease()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SubmitTask(func() {
			// 简单任务
			_ = 1 + 1
		})
	}
}

// BenchmarkSubmitTaskWithData 基准测试：带数据提交
func BenchmarkSubmitTaskWithData(b *testing.B) {
	InitCoroutinePool()
	defer CoroutineRelease()

	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SubmitTaskWithData(data, func(d interface{}) {
			// 简单任务
			_ = d
		})
	}
}

// BenchmarkSubmitTaskGeneric 基准测试：泛型提交
func BenchmarkSubmitTaskGeneric(b *testing.B) {
	InitCoroutinePool()
	defer CoroutineRelease()

	type Data struct {
		Key1 string
		Key2 int
	}

	data := Data{
		Key1: "value1",
		Key2: 123,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SubmitTaskGeneric(data, func(d Data) {
			// 简单任务
			_ = d
		})
	}
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	InitCoroutinePool()
	defer CoroutineRelease()

	t.Run("不可序列化的数据", func(t *testing.T) {
		// channel 不能被 JSON 序列化
		ch := make(chan int)

		err := SubmitTaskWithData(ch, func(d interface{}) {
			t.Error("不应该执行到这里")
		})

		if err == nil {
			t.Error("期望返回错误，但没有")
		}
	})

	t.Run("协程池未初始化", func(t *testing.T) {
		// 保存当前池
		oldPool := antsPool
		antsPool = nil

		err := SubmitTaskWithData(map[string]string{"key": "value"}, func(d interface{}) {})

		if err == nil {
			t.Error("期望返回错误，但没有")
		}

		// 恢复池
		antsPool = oldPool
	})
}

// ExampleSubmitTaskWithData 示例：使用 SubmitTaskWithData
func ExampleSubmitTaskWithData() {
	InitCoroutinePool()
	defer CoroutineRelease()

	data := map[string]interface{}{
		"userId": 123,
		"action": "login",
	}

	SubmitTaskWithData(data, func(d interface{}) {
		dataMap := d.(map[string]interface{})
		// 处理数据
		_ = dataMap
	})

	time.Sleep(time.Millisecond * 100) // 等待任务完成
}

// ExampleSubmitTaskGeneric 示例：使用泛型方法
func ExampleSubmitTaskGeneric() {
	InitCoroutinePool()
	defer CoroutineRelease()

	type User struct {
		ID   int
		Name string
	}

	user := User{ID: 123, Name: "test"}

	SubmitTaskGeneric(user, func(u User) {
		// 类型安全，无需类型断言
		_ = u.Name
	})

	time.Sleep(time.Millisecond * 100) // 等待任务完成
}
