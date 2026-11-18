package cache

import (
	"context"
	"testing"
	"time"
)

func mk(t *testing.T, capN int) *LRUcache {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return NewLRUcache(capN, time.Hour, ctx)
}

// список после Set a,b,c: c -> b -> a (a хвост)
func TestTailDeleteThenTouchNewTail(t *testing.T) {
	c := mk(t, 3)
	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	c.Set("c", []byte("3"))
	c.Delete("a") // удаляем хвост, остаются c -> b
	c.Get("b")    // b новый хвост: двигаем в голову
	c.Set("d", []byte("4"))
	c.Set("e", []byte("5")) // переполнение: должен вытеснить самый старый
	t.Logf("size=%d", c.Stats()["currentSize"])
	if s := c.Stats()["currentSize"]; s > 3 {
		t.Fatalf("capacity exceeded: %d", s)
	}
	// проверка согласованности списка и map
	n := 0
	for p := c.head; p != nil; p = p.next {
		n++
		if n > 100 {
			t.Fatal("цикл в списке")
		}
	}
	if n != len(c.items) {
		t.Fatalf("список=%d узлов, map=%d", n, len(c.items))
	}
}
