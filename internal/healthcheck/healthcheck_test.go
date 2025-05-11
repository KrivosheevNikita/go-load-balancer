package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loadbalancer/internal/loadbalancer"
)

func TestChecker_Check(t *testing.T) {
	srvOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srvOK.Close()
	srvBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srvBad.Close()

	bOK, _ := loadbalancer.NewBackend(srvOK.URL)
	bBad, _ := loadbalancer.NewBackend(srvBad.URL)
	bOK.SetAlive(false)
	bBad.SetAlive(true)

	c := New([]loadbalancer.Backend{bOK, bBad}, 10*time.Millisecond)
	c.check(bOK)
	if !bOK.Alive() {
		t.Error("expected healthy backend")
	}
	c.check(bBad)
	if bBad.Alive() {
		t.Error("expected bad backend")
	}
}

func TestChecker_StartStop(t *testing.T) {
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if count%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	b, _ := loadbalancer.NewBackend(srv.URL)
	c := New([]loadbalancer.Backend{b}, 20*time.Millisecond)
	c.Start()
	time.Sleep(100 * time.Millisecond)
	c.Stop()
	if count < 2 {
		t.Errorf("expected 2+ checks, got %d", count)
	}
}
