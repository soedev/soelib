package tenantdb

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/soedev/soelib/common/db/specialdb"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGetDbFromMapWithOpt(t *testing.T) {
	dbInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable password=%s application_name=go-soelib",
		"192.168.1.208", 31209, "soe", "soeoadb", "soesoft")
	config := specialdb.DbConfig{
		DBInfo:          dbInfo,
		Dialect:         "postgres",
		DBConfig:        gorm.Config{Logger: logger.Default.LogMode(logger.Info)},
		MaxIdleConns:    1,
		MaxOpenConns:    50,
		ConnMaxLifetime: time.Minute * 5,
	}
	CrmDb, err := specialdb.ConnDB(config)
	if err != nil {
		t.Fatal(err)
	}
	db, err := GetDbFromMapWithOpt("600002", CrmDb, &OptSQL{
		DBConfig:        gorm.Config{Logger: logger.Default.LogMode(logger.Info)},
		ApplicationName: "soe-lib",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec("select 1").Error
	if err != nil {
		t.Fatal(err)
	}
}

// TestConcurrentGetDbFromMap 测试高并发场景下获取同一租户连接的性能
func TestConcurrentGetDbFromMap(t *testing.T) {
	dbInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable password=%s application_name=go-soelib-test",
		"192.168.1.208", 31209, "soe", "soeoadb", "soesoft")
	config := specialdb.DbConfig{
		DBInfo:          dbInfo,
		Dialect:         "postgres",
		DBConfig:        gorm.Config{Logger: logger.Default.LogMode(logger.Error)},
		MaxIdleConns:    2,
		MaxOpenConns:    50,
		ConnMaxLifetime: time.Minute * 5,
	}
	CrmDb, err := specialdb.ConnDB(config)
	if err != nil {
		t.Fatal(err)
	}

	// 清理测试环境
	defer func() {
		UpdateMapV2("600002")
		StopHealthChecker()
	}()

	// 测试场景：100个并发请求同时获取同一租户的连接
	concurrency := 100
	tenantID := "600002"

	var wg sync.WaitGroup
	wg.Add(concurrency)

	// 记录开始时间
	startTime := time.Now()

	// 用于记录每个请求的耗时
	durations := make([]time.Duration, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			defer wg.Done()

			reqStart := time.Now()
			db, err := GetDbFromMap(tenantID, CrmDb, "soe-lib-test", false, logger.Error)
			reqDuration := time.Since(reqStart)
			durations[index] = reqDuration

			if err != nil {
				t.Errorf("并发请求 %d 失败: %v", index, err)
				return
			}

			// 执行一个简单查询验证连接可用
			if err := db.Exec("select 1").Error; err != nil {
				t.Errorf("并发请求 %d 执行查询失败: %v", index, err)
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	// 统计结果
	var totalDur time.Duration
	var maxDur time.Duration
	var minDur = time.Hour

	for _, d := range durations {
		totalDur += d
		if d > maxDur {
			maxDur = d
		}
		if d < minDur {
			minDur = d
		}
	}

	avgDur := totalDur / time.Duration(concurrency)

	t.Logf("并发测试结果 (并发数: %d, 租户: %s):", concurrency, tenantID)
	t.Logf("  总耗时: %v", totalDuration)
	t.Logf("  平均单次请求耗时: %v", avgDur)
	t.Logf("  最快请求耗时: %v", minDur)
	t.Logf("  最慢请求耗时: %v", maxDur)

	// 性能断言：平均耗时应该小于100ms（优化前可能达到2s）
	if avgDur > 100*time.Millisecond {
		t.Errorf("平均耗时过长: %v, 期望小于100ms", avgDur)
	}

	// 最慢请求也不应该超过500ms
	if maxDur > 500*time.Millisecond {
		t.Errorf("最慢请求耗时过长: %v, 期望小于500ms", maxDur)
	}
}

// BenchmarkGetDbFromMapConcurrent 基准测试：模拟高并发场景
func BenchmarkGetDbFromMapConcurrent(b *testing.B) {
	dbInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable password=%s application_name=go-soelib-bench",
		"192.168.1.208", 31209, "soe", "soeoadb", "soesoft")
	config := specialdb.DbConfig{
		DBInfo:          dbInfo,
		Dialect:         "postgres",
		DBConfig:        gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
		MaxIdleConns:    2,
		MaxOpenConns:    50,
		ConnMaxLifetime: time.Minute * 5,
	}
	CrmDb, err := specialdb.ConnDB(config)
	if err != nil {
		b.Fatal(err)
	}

	defer func() {
		UpdateMapV2("600002")
		StopHealthChecker()
	}()

	tenantID := "600002"

	// 预热：先创建连接
	_, _ = GetDbFromMap(tenantID, CrmDb, "soe-lib-bench", false, logger.Silent)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			db, err := GetDbFromMap(tenantID, CrmDb, "soe-lib-bench", false, logger.Silent)
			if err != nil {
				b.Errorf("获取连接失败: %v", err)
				continue
			}
			// 模拟实际使用
			_ = db.Exec("select 1")
		}
	})
}
