package hw04lrucache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		l := NewList()

		require.Equal(t, 0, l.Len())
		require.Nil(t, l.Front())
		require.Nil(t, l.Back())
	})

	t.Run("complex", func(t *testing.T) {
		l := NewList()

		l.PushFront(10) // [10]
		l.PushBack(20)  // [10, 20]
		l.PushBack(30)  // [10, 20, 30]
		require.Equal(t, 3, l.Len())

		middle := l.Front().Next // 20
		l.Remove(middle)         // [10, 30]
		require.Equal(t, 2, l.Len())

		for i, v := range [...]int{40, 50, 60, 70, 80} {
			if i%2 == 0 {
				l.PushFront(v)
			} else {
				l.PushBack(v)
			}
		} // [80, 60, 40, 10, 30, 50, 70]

		require.Equal(t, 7, l.Len())
		require.Equal(t, 80, l.Front().Value)
		require.Equal(t, 70, l.Back().Value)

		l.MoveToFront(l.Front()) // [80, 60, 40, 10, 30, 50, 70]
		l.MoveToFront(l.Back())  // [70, 80, 60, 40, 10, 30, 50]

		elems := make([]int, 0, l.Len())
		for i := l.Front(); i != nil; i = i.Next {
			elems = append(elems, i.Value.(int))
		}
		require.Equal(t, []int{70, 80, 60, 40, 10, 30, 50}, elems)
	})

	// Дополнительные тесты для проверки граничных условий (edge cases)
	t.Run("remove edge cases", func(t *testing.T) {
		l := NewList()

		// 1. Удаление единственного элемента
		item := l.PushBack(100)
		l.Remove(item)
		require.Equal(t, 0, l.Len())
		require.Nil(t, l.Front())
		require.Nil(t, l.Back())

		// 2. Удаление первого элемента (Front)
		l.PushBack(10)      // [10]
		l.PushBack(20)      // [10, 20]
		l.PushBack(30)      // [10, 20, 30]
		l.Remove(l.Front()) // [20, 30]
		require.Equal(t, 2, l.Len())
		require.Equal(t, 20, l.Front().Value)
		require.Equal(t, 30, l.Back().Value)
		require.Nil(t, l.Front().Prev) // Проверка инварианта nil <- front

		// 3. Удаление последнего элемента (Back)
		l.Remove(l.Back()) // [20]
		require.Equal(t, 1, l.Len())
		require.Equal(t, 20, l.Front().Value)
		require.Equal(t, 20, l.Back().Value)
		require.Nil(t, l.Back().Next) // Проверка инварианта back -> nil
	})

	t.Run("move to front edge cases", func(t *testing.T) {
		l := NewList()

		// 1. MoveToFront для списка из одного элемента
		item := l.PushBack(1)
		l.MoveToFront(item)
		require.Equal(t, 1, l.Len())
		require.Equal(t, 1, l.Front().Value)
		require.Equal(t, 1, l.Back().Value)
		require.Nil(t, l.Front().Prev)
		require.Nil(t, l.Back().Next)

		// 2. MoveToFront для списка из двух элементов (перемещение Back в Front)
		l.PushBack(2)           // [1, 2]
		l.MoveToFront(l.Back()) // [2, 1]
		require.Equal(t, 2, l.Len())
		require.Equal(t, 2, l.Front().Value)
		require.Equal(t, 1, l.Back().Value)
		require.Nil(t, l.Front().Prev)
		require.Nil(t, l.Back().Next)
	})
}
