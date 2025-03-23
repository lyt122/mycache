package lru

import (
	"container/list"
	"sync"
)

type Cache struct {
	maxBytes  int64
	nBytes    int64
	list      *list.List
	cache     map[string]*list.Element
	OnEvicted func(key string, val Value)

	mu       sync.Mutex
	cond     *sync.Cond
	stopChan chan struct{}
}

type entry struct {
	key string
	val Value
}

type Value interface {
	Len() int
}

func New(maxBytes int64, onEvict func(key string, val Value)) *Cache {
	c := &Cache{
		maxBytes:  maxBytes,
		list:      list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvict,
		stopChan:  make(chan struct{}),
	}
	c.cond = sync.NewCond(&c.mu)
	go c.monitor()
	return c
}

func (c *Cache) monitor() {
	for {
		c.mu.Lock()
		// 如果不需要清理，就等待信号
		for c.nBytes <= c.maxBytes {
			c.cond.Wait()
			// 退出 goroutine
			select {
			case <-c.stopChan:
				c.mu.Unlock()
				return
			default:
			}
		}

		// 进行清理
		for c.nBytes > c.maxBytes {
			c.RemoveOldest()
		}

		c.mu.Unlock()
	}
}

func (c *Cache) Get(key string) (Value, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.cache[key]; ok {
		c.list.MoveToFront(element)
		kv := element.Value.(*entry)
		return kv.val, true
	}
	return nil, false
}

func (c *Cache) RemoveOldest() {
	element := c.list.Back()
	if element != nil {
		kv := element.Value.(*entry)
		delete(c.cache, kv.key)
		c.list.Remove(element)
		c.nBytes -= int64(len(kv.key)) + int64(kv.val.Len())

		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.val)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.cache[key]; ok {
		c.list.MoveToFront(element)
		kv := element.Value.(*entry)
		c.nBytes -= int64(len(kv.key)) + int64(kv.val.Len())
		kv.val = value
	} else {
		ele := c.list.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}

	c.cond.Signal()
}

func (c *Cache) Stop() {
	close(c.stopChan)
	c.cond.Signal()
}

func (c *Cache) Len() int {
	return c.list.Len()
}
