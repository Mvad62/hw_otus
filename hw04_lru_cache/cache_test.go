package hw04lrucache

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		c := NewCache(10)

		_, ok := c.Get("aaa")
		require.False(t, ok)

		_, ok = c.Get("bbb")
		require.False(t, ok)
	})

	t.Run("simple", func(t *testing.T) {
		c := NewCache(5)

		wasInCache := c.Set("aaa", 100)
		require.False(t, wasInCache)

		wasInCache = c.Set("bbb", 200)
		require.False(t, wasInCache)

		val, ok := c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 100, val)

		val, ok = c.Get("bbb")
		require.True(t, ok)
		require.Equal(t, 200, val)

		wasInCache = c.Set("aaa", 300)
		require.True(t, wasInCache)

		val, ok = c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 300, val)

		val, ok = c.Get("ccc")
		require.False(t, ok)
		require.Nil(t, val)
	})

	t.Run("purge logic", func(t *testing.T) {
		t.Run("eviction by capacity", func(t *testing.T) {
			// Тест на логику выталкивания элементов из-за размера очереди
			c := NewCache(3)
			c.Set("a", 1)
			c.Set("b", 2)
			c.Set("c", 3)
			c.Set("d", 4) // "a" должен быть вытолкнут, так как он был добавлен первым и к нему не обращались

			_, ok := c.Get("a")
			require.False(t, ok, "элемент 'a' должен быть вытолкнут из-за переполнения емкости")

			val, ok := c.Get("b")
			require.True(t, ok)
			require.Equal(t, 2, val)

			val, ok = c.Get("c")
			require.True(t, ok)
			require.Equal(t, 3, val)

			val, ok = c.Get("d")
			require.True(t, ok)
			require.Equal(t, 4, val)
		})
		t.Run("eviction by usage (LRU)", func(t *testing.T) {
			// Тест на логику выталкивания давно используемых элементов
			c := NewCache(3)
			c.Set("a", 1)
			c.Set("b", 2)
			c.Set("c", 3)

			// Обращаемся к "a", делая его недавно использованным (перемещаем в начало очереди)
			_, ok := c.Get("a")
			require.True(t, ok)

			// Обновляем "b", также делая его недавно использованным (перемещаем в начало очереди)
			wasInCache := c.Set("b", 20)
			require.True(t, wasInCache)

			// Теперь "c" является наименее недавно использованным (LRU), так как к "a" и "b" обращались после их добавления
			c.Set("d", 4) // "c" должен быть вытолкнут

			_, ok = c.Get("c")
			require.False(t, ok, "элемент 'c' должен быть вытолкнут, так как к нему не обращались")

			val, ok := c.Get("a")
			require.True(t, ok)
			require.Equal(t, 1, val)

			val, ok = c.Get("b")
			require.True(t, ok)
			require.Equal(t, 20, val)

			val, ok = c.Get("d")
			require.True(t, ok)
			require.Equal(t, 4, val)
		})
	})
	t.Run("clear", func(t *testing.T) {
		c := NewCache(2)
		c.Set("a", 1)
		c.Set("b", 2)

		c.Clear()

		_, ok := c.Get("a")
		require.False(t, ok, "кэш должен быть пуст после очистки")

		_, ok = c.Get("b")
		require.False(t, ok, "кэш должен быть пуст после очистки")

		// Проверяем, что после очистки можно снова корректно добавлять элементы
		c.Set("c", 3)
		val, ok := c.Get("c")
		require.True(t, ok)
		require.Equal(t, 3, val)
	})
}

func TestCacheMultithreading(t *testing.T) {
	c := NewCache(10)
	wg := &sync.WaitGroup{}

	// Уменьшаем количество ключей для создания высокой конкуренции (contention)
	// за одни и те же элементы списка (частые вызовы MoveToFront)
	const numKeys = 50
	const iterations = 100_000

	wg.Add(2)

	// Горутина 1: постоянно перезаписывает ограниченный набор ключей
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			// Используем остаток от деления, чтобы ключи гарантированно повторялись
			key := Key(strconv.Itoa(i % numKeys))
			c.Set(key, i)
		}
	}()

	// Горутина 2: постоянно читает тот же ограниченный набор ключей
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			key := Key(strconv.Itoa(rand.Intn(numKeys)))
			// Мы не проверяем результат здесь, чтобы не замедлять тест,
			// но сам факт вызова Get создает гонку за мьютекс и указатели списка
			c.Get(key)
		}
	}()

	wg.Wait()

	// 1. Проверяем, что размер кэша не превышает емкость (защита от утечек в map)
	// Для этого нам нужно привести интерфейс к конкретной реализации,
	// либо добавить метод Len() в интерфейс Cache.
	// Допустим, мы добавили Len() в интерфейс Cache:
	// require.LessOrEqual(t, c.Len(), 10, "Размер кэша не должен превышать емкость")

	// 2. Проверяем, что оставшиеся в кэше элементы имеют корректные значения
	// (они должны быть не nil, так как мы только что их активно писали)
	for i := 0; i < numKeys; i++ {
		key := Key(strconv.Itoa(i))
		val, ok := c.Get(key)
		if ok {
			// Если элемент есть, он должен быть целым числом (наш тип значения)
			require.IsType(t, 0, val, "Значение в кэше должно быть int")
		}
	}
}
