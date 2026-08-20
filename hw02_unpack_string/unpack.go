package hw02unpackstring

import (
	"errors"
)

var ErrInvalidString = errors.New("invalid string")

// Unpack распаковывает строку с поддержкой экранирования через \.
func Unpack(s string) (string, error) {
	runes := []rune(s)
	result := make([]rune, 0, len(runes))
	prevWasDigit := false

	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		// Обработка экранирования: заэкранировать можно только цифру или слэш
		if ch == '\\' {
			if i+1 >= len(runes) {
				return "", ErrInvalidString
			}
			next := runes[i+1]
			if next != '\\' && !(next >= '0' && next <= '9') {
				return "", ErrInvalidString
			}
			result = append(result, next)
			i++ // пропускаем экранированный символ
			prevWasDigit = false
			continue
		}

		// Если текущий символ – цифра от '0' до '9'
		if ch >= '0' && ch <= '9' {
			// Цифра не может быть первой, и цифры не могут идти подряд
			if len(result) == 0 || prevWasDigit {
				return "", ErrInvalidString
			}

			count := int(ch - '0')
			last := result[len(result)-1]

			if count == 0 {
				// Удаляем последнюю руну
				result = result[:len(result)-1]
			} else {
				// Добавляем count повторений последней руны
				for j := 0; j < count-1; j++ {
					result = append(result, last)
				}
			}

			prevWasDigit = true
			continue
		}

		// Обычный символ (не цифра и не обратный слеш)
		result = append(result, ch)
		prevWasDigit = false
	}

	return string(result), nil
}
