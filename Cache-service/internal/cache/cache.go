package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Node struct {
	key  string
	data []byte

	next *Node
	prev *Node

	expireAt time.Time
}

type LRUcache struct {
	capacity int //	Size of cache
	items    map[string]*Node

	head *Node
	tail *Node

	ttl       time.Duration
	mu        sync.RWMutex
	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64 // Count of times to work LRU
}

func NewLRUcache(capacity int, ttl time.Duration, ctx context.Context) *LRUcache {
	lc := &LRUcache{capacity: capacity, items: make(map[string]*Node, capacity), ttl: ttl}
	lc.startTTLWorker(ctx)
	return lc
}

func (lc *LRUcache) moveToFront(ptr *Node) {
	if ptr == lc.head { // Проверка на то что итак в голове
		return
	}

	if ptr.next != nil { //Есть эдемент и справа и слева
		ptr.prev.next = ptr.next
		ptr.next.prev = ptr.prev
		ptr.next = lc.head
		lc.head.prev = ptr
		ptr.prev = nil
		lc.head = ptr
	} else { //Мы в конце списка хвост в голову
		lc.tail = ptr.prev  // 1. читаем ptr.prev — сохраняем новый хвост
		ptr.prev.next = nil // 2. читаем ptr.prev — отрезаем от списка
		ptr.prev = nil      // 3. только теперь обнуляем
		ptr.next = lc.head
		lc.head.prev = ptr
		lc.head = ptr
	}
}

func (lc *LRUcache) removeNode(ptr *Node) {
	delete(lc.items, ptr.key)
	if ptr == lc.head && ptr == lc.tail {
		lc.head = nil
		lc.tail = nil
	} else if ptr == lc.head && ptr != lc.tail {
		lc.head = ptr.next
		lc.head.prev = nil
		ptr.next = nil
	} else if ptr != lc.head && ptr == lc.tail {
		lc.tail = ptr.prev
		ptr.prev = nil
	} else {
		ptr.prev.next = ptr.next
		ptr.next.prev = ptr.prev
		ptr.next = nil
		ptr.prev = nil
	}
}
func (lc *LRUcache) addToFront(ptr *Node) {
	ptr.next = lc.head
	ptr.prev = nil
	if lc.head != nil {
		lc.head.prev = ptr
	}
	lc.head = ptr
	if lc.tail == nil { // список был пустой
		lc.tail = ptr
	}
}

func (lc *LRUcache) evicTail() {
	if lc.tail == nil {
		return
	}

	delete(lc.items, lc.tail.key)

	if lc.tail == lc.head { // Обработали случай когда у нас голова это хвост
		lc.tail = nil
		lc.head = nil
		return
	}

	oldTail := lc.tail
	lc.tail = oldTail.prev // сначала сохраняем
	lc.tail.next = nil     // потом отрезаем
	oldTail.prev = nil

}

func (lc *LRUcache) Get(key string) ([]byte, bool) {
	lc.mu.Lock()
	val, ok := lc.items[key]
	if !ok {
		lc.mu.Unlock()
		lc.misses.Add(1)
		return nil, false
	}

	if time.Now().After(val.expireAt) {
		lc.removeNode(val)
		lc.mu.Unlock()
		lc.misses.Add(1)
		return nil, false
	}

	lc.moveToFront(val)
	result := make([]byte, len(val.data))
	copy(result, val.data)
	lc.mu.Unlock()
	lc.hits.Add(1)
	return result, true
}

func (lc *LRUcache) Set(key string, data []byte) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	_, ok := lc.items[key]
	newNode := &Node{data: data, key: key, expireAt: time.Now().Add(lc.ttl)}
	if ok {
		lc.removeNode(lc.items[key])
		lc.items[key] = newNode
		lc.addToFront(newNode)
		return
	}

	if len(lc.items) == lc.capacity {
		lc.evicTail()
		lc.evictions.Add(1)
	}
	lc.items[key] = newNode
	lc.addToFront(newNode)
}

func (lc *LRUcache) Delete(key string) bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	val, ok := lc.items[key]
	if ok {
		lc.removeNode(val)
		return true
	}

	return false
}

func (lc *LRUcache) Stats() map[string]int64 {
	lc.mu.Lock()
	size := int64(len(lc.items))
	lc.mu.Unlock()
	return map[string]int64{
		"currentSize": size,
		"hits":        lc.hits.Load(),
		"misses":      lc.misses.Load(),
		"evictions":   lc.evictions.Load(),
	}
}

func (lc *LRUcache) startTTLWorker(ctx context.Context) {
	ticker := time.NewTicker(lc.ttl / 2)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				lc.mu.Lock()
				for _, node := range lc.items {
					if time.Now().After(node.expireAt) {
						lc.removeNode(node)
					}
				}
				lc.mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
}
