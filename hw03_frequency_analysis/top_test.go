package hw03frequencyanalysis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Change to true if needed.
var taskWithAsteriskIsCompleted = true

var text = `Как видите, он  спускается  по  лестнице  вслед  за  своим
	другом   Кристофером   Робином,   головой   вниз,  пересчитывая
	ступеньки собственным затылком:  бум-бум-бум.  Другого  способа
	сходить  с  лестницы  он  пока  не  знает.  Иногда ему, правда,
		кажется, что можно бы найти какой-то другой способ, если бы  он
	только   мог   на  минутку  перестать  бумкать  и  как  следует
	сосредоточиться. Но увы - сосредоточиться-то ему и некогда.
		Как бы то ни было, вот он уже спустился  и  готов  с  вами
	познакомиться.
	- Винни-Пух. Очень приятно!
		Вас,  вероятно,  удивляет, почему его так странно зовут, а
	если вы знаете английский, то вы удивитесь еще больше.
		Это необыкновенное имя подарил ему Кристофер  Робин.  Надо
	вам  сказать,  что  когда-то Кристофер Робин был знаком с одним
	лебедем на пруду, которого он звал Пухом. Для лебедя  это  было
	очень   подходящее  имя,  потому  что  если  ты  зовешь  лебедя
	громко: "Пу-ух! Пу-ух!"- а он  не  откликается,  то  ты  всегда
	можешь  сделать вид, что ты просто понарошку стрелял; а если ты
	звал его тихо, то все подумают, что ты  просто  подул  себе  на
	нос.  Лебедь  потом  куда-то делся, а имя осталось, и Кристофер
	Робин решил отдать его своему медвежонку, чтобы оно не  пропало
	зря.
		А  Винни - так звали самую лучшую, самую добрую медведицу
	в  зоологическом  саду,  которую  очень-очень  любил  Кристофер
	Робин.  А  она  очень-очень  любила  его. Ее ли назвали Винни в
	честь Пуха, или Пуха назвали в ее честь - теперь уже никто  не
	знает,  даже папа Кристофера Робина. Когда-то он знал, а теперь
	забыл.
		Словом, теперь мишку зовут Винни-Пух, и вы знаете почему.
		Иногда Винни-Пух любит вечерком во что-нибудь поиграть,  а
	иногда,  особенно  когда  папа  дома,  он больше любит тихонько
	посидеть у огня и послушать какую-нибудь интересную сказку.
		В этот вечер...`

func TestTop10(t *testing.T) {
	t.Run("no words in empty string", func(t *testing.T) {
		require.Len(t, Top10(""), 0)
	})

	t.Run("positive test", func(t *testing.T) {
		if taskWithAsteriskIsCompleted {
			expected := []string{
				"а",         // 8
				"он",        // 8
				"и",         // 6
				"ты",        // 5
				"что",       // 5
				"в",         // 4
				"его",       // 4
				"если",      // 4
				"кристофер", // 4
				"не",        // 4
			}
			require.Equal(t, expected, Top10(text))
		} else {
			expected := []string{
				"он",        // 8
				"а",         // 6
				"и",         // 6
				"ты",        // 5
				"что",       // 5
				"-",         // 4
				"Кристофер", // 4
				"если",      // 4
				"не",        // 4
				"то",        // 4
			}
			require.Equal(t, expected, Top10(text))
		}
	})
}

func TestTop10_EdgeCases(t *testing.T) {
	// Таблица тестовых сценариев
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "only invalid words (spaces, punctuation, single hyphen)",
			input: "   -  ,,,  ...  \n\t  '  ",
			// Ожидаем пустой слайс, так как все символы будут отсеяны
			expected: []string{},
		},
		{
			name:  "case insensitivity and edge punctuation trimming",
			input: "Нога! нога, 'НОГА' ...нога...",
			// Все варианты должны схлопнуться в одно слово "нога"
			expected: []string{"нога"},
		},
		{
			name:  "hyphen rules: single hyphen ignored, multiple kept",
			input: "- -- --- -a- a- -a",
			// "-" игнорируется. Остальные сортируются лексикографически.
			// Порядок в ASCII: '-' (45), 'a' (97). Поэтому "-a" < "-a-" < "a-"
			expected: []string{"--", "---", "-a", "-a-", "a-"},
		},
		{
			name:  "hyphenated words are distinct from non-hyphenated",
			input: "какой-то какойто",
			// Это разные слова, частота у каждого 1, сортировка лексикографическая
			expected: []string{"какой-то", "какойто"},
		},
		{
			name:  "punctuation inside word is NOT trimmed",
			input: "dog...cat dogcat",
			// "dog...cat" не разбивается, так как разделитель только пробел.
			// '.' (46) идет раньше 'c' (99), поэтому "dog...cat" будет первым.
			expected: []string{"dog...cat", "dogcat"},
		},
		{
			name:  "lexicographical tie-breaking for exactly 10 words",
			input: "j i h g f e d c b a",
			// Все слова встречаются 1 раз. Должны быть отсортированы по алфавиту.
			expected: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
		},
		{
			name:  "lexicographical tie-breaking with MORE than 10 words (the asterisk task)",
			input: "z y x w v u t s r q p o n m l k j i h g f e d c b a",
			// 25 слов с частотой 1. Должны вернуться первые 10 в алфавитном порядке.
			expected: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
		},
		{
			name:  "mixed frequencies with tie-breaking",
			input: "apple banana apple cherry banana date apple",
			// apple: 3, banana: 2, cherry: 1, date: 1
			// cherry и date имеют частоту 1, "cherry" < "date" лексикографически.
			expected: []string{"apple", "banana", "cherry", "date"},
		},
		{
			name:  "unicode and numbers handling",
			input: "123 123! test-1 test-1",
			// Цифры сохраняются. "123" встречается 2 раза, "test-1" 2 раза.
			// '1' (49) < 't' (116), поэтому "123" будет первым.
			expected: []string{"123", "test-1"},
		},
		{
			name:     "emoji as part of words",
			input:    "😊 apple 😊 banana! 😊",
			expected: []string{"😊", "apple", "banana"},
		},
		{
			name:     "emoji with surrounding punctuation trimmed",
			input:    "😀! '😀' ...😀...",
			expected: []string{"😀"},
		},
		{
			name:     "lexicographical order with emojis",
			input:    "😀 😁 😃",
			expected: []string{"😀", "😁", "😃"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// require.Equal отлично сравнивает слайсы, включая проверку порядка элементов
			// и корректно обрабатывает случай, когда ожидаемый слайс пуст, а результат nil или []string{}
			require.Equal(t, tt.expected, Top10(tt.input))
		})
	}
}
