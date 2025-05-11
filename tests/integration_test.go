package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"loadbalancer/internal/loadbalancer"
	"loadbalancer/internal/proxy"
	"loadbalancer/internal/ratelimiter"
	"loadbalancer/internal/server"
)

func TestLoadBalancerIntegration(t *testing.T) {
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "backend1")
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "backend2")
	}))
	defer backend2.Close()

	b1, err := loadbalancer.NewBackend(backend1.URL)
	if err != nil {
		t.Fatalf("NewBackend backend1 fail: %v", err)
	}
	b2, err := loadbalancer.NewBackend(backend2.URL)
	if err != nil {
		t.Fatalf("NewBackend backend2 fail: %v", err)
	}
	bs := []loadbalancer.Backend{b1, b2}

	sel := loadbalancer.NewRoundRobin(bs)
	px := proxy.New(sel)
	rl := ratelimiter.NewStore(1000, 1000, nil)
	handler := server.BuildHandler(px, rl.Middleware)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	total := 100
	results := map[string]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(total)

	for i := 0; i != total; i++ {
		go func() {
			defer wg.Done()
			resp, err := client.Get(ts.URL)
			if err != nil {
				t.Logf("request error: %v", err)
				return
			}
			defer resp.Body.Close()
			buf := make([]byte, len("backend1"))
			n, _ := resp.Body.Read(buf)
			key := string(buf[:n])
			mu.Lock()
			results[key]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if results["backend1"] == 0 || results["backend2"] == 0 {
		t.Errorf("expected both backends to receive traffic, got: %v", results)
	}
}

func Benchmark(b *testing.B) {
	client := &http.Client{Timeout: 2 * time.Second}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = client.Get("http://localhost:8080/")
		}
	})
}
