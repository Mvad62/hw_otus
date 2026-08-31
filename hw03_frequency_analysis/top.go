package hw03frequencyanalysis

import (
	"regexp"
	"sort"
	"strings"
)

var trimRegex = regexp.MustCompile(`^[^\P{P}-]+|[^\P{P}-]+$`)

// Top10 возвращает до 10 самых часто встречающихся слов в строке s.
func Top10(s string) []string {
	words := strings.Fields(s)
	freq := make(map[string]int, len(words)/2) // Начальная емкость словаря

	for _, w := range words {
		// Приводим к нижнему регистру
		w = strings.ToLower(w)

		// Удаляем знаки препинания по краям, кроме дефиса
		w = trimRegex.ReplaceAllString(w, "")

		// Пропускаем пустые строки и одиночный дефис
		if w == "" || w == "-" {
			continue
		}

		freq[w]++
	}

	type wordCount struct {
		word  string
		count int
	}

	counts := make([]wordCount, 0, len(freq))
	for w, c := range freq {
		counts = append(counts, wordCount{word: w, count: c})
	}

	// Сортировка: сначала по убыванию частоты, при равенстве — лексикографически
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].count != counts[j].count {
			return counts[i].count > counts[j].count
		}
		return counts[i].word < counts[j].word
	})

	// Ограничиваем результат 10 элементами
	if len(counts) > 10 {
		counts = counts[:10]
	}

	// Формируем итоговый слайс строк
	res := make([]string, len(counts))
	for i, wc := range counts {
		res[i] = wc.word
	}

	return res
}
