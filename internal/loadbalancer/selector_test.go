package loadbalancer

import (
	"net/url"
	"testing"
)

type mockBackend struct {
	u     string
	alive bool
	conns int64
}

func (f *mockBackend) URL() *url.URL {
	u, _ := url.Parse(f.u)
	return u
}
func (m *mockBackend) Alive() bool     { return m.alive }
func (m *mockBackend) SetAlive(v bool) { m.alive = v }
func (m *mockBackend) Inc()            { m.conns++ }
func (m *mockBackend) Done()           { m.conns-- }
func (m *mockBackend) Conns() int64    { return m.conns }

func TestRoundRobin(t *testing.T) {
	bs := []Backend{
		&mockBackend{"b1", true, 0},
		&mockBackend{"b2", true, 0},
		&mockBackend{"b3", true, 0},
	}
	rr := NewRoundRobin(bs)
	order := []string{"b1", "b2", "b3", "b1", "b2", "b3", "b1"}
	for _, want := range order {
		b := rr.Next()
		if b.URL().String() != want {
			t.Errorf("got %s, want %s", b.URL(), want)
		}
	}
}

func TestLeastConnections(t *testing.T) {
	bs := []Backend{
		&mockBackend{"b1", true, 5},
		&mockBackend{"b2", true, 2},
		&mockBackend{"b3", true, 8},
	}
	lc := NewLeastConnections(bs)
	b := lc.Next()
	if b.URL().String() != "b2" {
		t.Errorf("got %s, want b2", b.URL())
	}

	bs[1].SetAlive(false)
	b2 := lc.Next()
	if b2.URL().String() != "b1" {
		t.Errorf("got %s, want b1", b2.URL())
	}
}
