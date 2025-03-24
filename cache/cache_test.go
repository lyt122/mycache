package cache

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

func TestGetConcurrent(t *testing.T) {
	db := map[string]string{"Tom": "630", "Jack": "589"} // 测试数据
	loadCounts := sync.Map{}                             // 替换为并发安全的 sync.Map
	gee := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			time.Sleep(10 * time.Millisecond) // 模拟慢数据库
			if v, ok := db[key]; ok {
				loadCounts.Store(key, 1) // 记录数据库加载次数
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	// 并发测试：10个协程同时请求相同 Key
	var wg sync.WaitGroup
	wg.Add(10)
	start := time.Now()
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			if v, err := gee.Get("Tom"); err != nil || v.String() != "630" {
				t.Error("failed to get value of Tom")
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	// 验证：数据库只应被加载 1 次（防击穿）
	if cnt, _ := loadCounts.Load("Tom"); cnt != 1 {
		t.Fatalf("cache breakdown: expected 1 DB load, got %d", cnt)
	}

	// 输出性能数据
	t.Logf("Concurrent Get took: %v (QPS: %.1f)", elapsed, 10/elapsed.Seconds())
}

func BenchmarkGet(b *testing.B) {
	db := map[string]string{"Tom": "630", "Jack": "589"}
	gee := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gee.Get("Tom") // 基准测试缓存命中性能
	}
}

func BenchmarkGetParallel(b *testing.B) {
	gee := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			time.Sleep(10 * time.Millisecond) // 模拟数据库延迟
			return []byte("630"), nil
		}))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gee.Get("Tom") // 高并发压测
		}
	})
}
