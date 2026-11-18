package cache

import (
	"fmt"
	"math/rand"
	"testing"
)

// эталонная модель LRU на срезе (без TTL)
type model struct {
	cap  int
	keys []string // [0] = самый свежий
	vals map[string]string
}

func (m *model) idx(k string) int {
	for i, x := range m.keys {
		if x == k {
			return i
		}
	}
	return -1
}
func (m *model) touch(k string) {
	i := m.idx(k)
	m.keys = append(m.keys[:i], m.keys[i+1:]...)
	m.keys = append([]string{k}, m.keys...)
}

func TestRandomVsModel(t *testing.T) {
	bad, panics := 0, 0
	const runs = 2000
	for seed := 0; seed < runs; seed++ {
		r := rand.New(rand.NewSource(int64(seed)))
		c := mk(t, 4)
		m := &model{cap: 4, vals: map[string]string{}}
		func() {
			defer func() {
				if recover() != nil {
					panics++
					bad++
				}
			}()
			for i := 0; i < 60; i++ {
				k := fmt.Sprintf("k%d", r.Intn(7))
				switch r.Intn(3) {
				case 0:
					c.Set(k, []byte("v"))
					if m.idx(k) >= 0 {
						m.touch(k)
					} else {
						if len(m.keys) == m.cap {
							last := m.keys[len(m.keys)-1]
							m.keys = m.keys[:len(m.keys)-1]
							delete(m.vals, last)
						}
						m.keys = append([]string{k}, m.keys...)
						m.vals[k] = "v"
					}
				case 1:
					_, ok := c.Get(k)
					if ok != (m.idx(k) >= 0) {
						bad++
						return
					}
					if ok {
						m.touch(k)
					}
				default:
					ok := c.Delete(k)
					if ok != (m.idx(k) >= 0) {
						bad++
						return
					}
					if ok {
						i := m.idx(k)
						m.keys = append(m.keys[:i], m.keys[i+1:]...)
						delete(m.vals, k)
					}
				}
				// список должен совпадать с map
				n := 0
				for p := c.head; p != nil && n <= 20; p = p.next {
					n++
				}
				if n != len(c.items) || len(c.items) > 4 {
					bad++
					return
				}
			}
		}()
	}
	t.Logf("запусков: %d, с расхождением: %d (из них паник: %d)", runs, bad, panics)
	if bad > 0 {
		t.Fail()
	}
}
