package hw04lrucache

import "sync"

type Key string

type Cache interface {
	Set(key Key, value interface{}) bool
	Get(key Key) (interface{}, bool)
	Clear()
}

type lruCache struct {
	// Cache // Remove me after realization.

	capacity int
	queue    List
	items    map[Key]*ListItem
	mu       sync.Mutex
}

type cacheItem struct {
	key   Key
	value interface{}
}

func NewCache(capacity int) Cache {
	return &lruCache{
		capacity: capacity,
		queue:    NewList(),
		items:    make(map[Key]*ListItem, capacity),
	}
}

// Set добавляет или обновляет значение в кэше.
func (c *lruCache) Set(key Key, value interface{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.items[key]; ok {
		// Элемент существует: обновляем значение и перемещаем в начало
		item.Value.(*cacheItem).value = value
		c.queue.MoveToFront(item)
		return true
	}

	// Элемента нет: добавляем новый в начало списка и в словарь
	elem := &cacheItem{key: key, value: value}
	listItem := c.queue.PushFront(elem)
	c.items[key] = listItem

	// Если превысили емкость, удаляем наименее недавно использованный элемент
	if c.queue.Len() > c.capacity {
		backItem := c.queue.Back()
		c.queue.Remove(backItem)
		delete(c.items, backItem.Value.(*cacheItem).key)
	}

	return false
}

// Get получает значение из кэша по ключу.
func (c *lruCache) Get(key Key) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	// Перемещаем использованный элемент в начало очереди
	c.queue.MoveToFront(item)
	return item.Value.(*cacheItem).value, true
}

// Clear очищает кэш.
func (c *lruCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.queue = NewList()
	c.items = make(map[Key]*ListItem)
}
