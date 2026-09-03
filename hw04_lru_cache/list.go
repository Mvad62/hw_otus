package hw04lrucache

type List interface {
	Len() int
	Front() *ListItem
	Back() *ListItem
	PushFront(v interface{}) *ListItem
	PushBack(v interface{}) *ListItem
	Remove(i *ListItem)
	MoveToFront(i *ListItem)
}

type ListItem struct {
	Value interface{}
	Next  *ListItem
	Prev  *ListItem
}

type list struct {
	front *ListItem
	back  *ListItem
	len   int
}

func NewList() List {
	return new(list)
}

func (l *list) Len() int {
	return l.len
}

func (l *list) Front() *ListItem {
	return l.front
}

func (l *list) Back() *ListItem {
	return l.back
}

func (l *list) PushFront(v interface{}) *ListItem {
	item := &ListItem{Value: v} // item.Prev и item.Next по умолчанию равны nil
	if l.len == 0 {
		l.front = item
		l.back = item
	} else {
		item.Next = l.front
		l.front.Prev = item
		l.front = item // Новый front, его Prev уже равен nil
	}
	l.len++
	return item
}

func (l *list) PushBack(v interface{}) *ListItem {
	item := &ListItem{Value: v} // item.Prev и item.Next по умолчанию равны nil
	if l.len == 0 {
		l.front = item
		l.back = item
	} else {
		item.Prev = l.back
		l.back.Next = item
		l.back = item // Новый back, его Next уже равен nil
	}
	l.len++
	return item
}

func (l *list) Remove(i *ListItem) {
	// Отсоединяем от предыдущего элемента
	if i.Prev != nil {
		i.Prev.Next = i.Next
	} else {
		l.front = i.Next
	}

	// Отсоединяем от следующего элемента
	if i.Next != nil {
		i.Next.Prev = i.Prev
	} else {
		l.back = i.Prev
	}

	// Полностью обнуляем указатели удаляемого элемента
	i.Prev = nil
	i.Next = nil
	l.len--
}

func (l *list) MoveToFront(i *ListItem) {
	if l.front == i {
		return // Уже в начале
	}

	// 1. Отсоединяем элемент от его текущего места
	if i.Prev != nil {
		i.Prev.Next = i.Next
	}
	if i.Next != nil {
		i.Next.Prev = i.Prev
	} else {
		// Если элемент был последним, обновляем указатель back.
		// У нового back (i.Prev) поле Next уже стало nil благодаря i.Prev.Next = i.Next выше.
		l.back = i.Prev
	}

	// 2. Вставляем элемент в начало
	i.Prev = nil // Явно гарантируем, что у нового front Prev == nil
	i.Next = l.front
	l.front.Prev = i
	l.front = i
}
