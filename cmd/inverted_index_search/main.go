package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"oip-course/internal/models"
	"os"
	"sort"
	"strings"

	"github.com/aaaton/golem/v4"
	"github.com/aaaton/golem/v4/dicts/ru"
)

var lemmatizer *golem.Lemmatizer

func init() {
	var err error
	lemmatizer, err = golem.New(ru.New())
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	index, err := loadInvertedIndex("inverted_index.json")
	if err != nil {
		log.Fatal(err)
	}

	// Создание сканера для чтения пользовательского ввода
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter your query:")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		query := strings.TrimSpace(scanner.Text())
		if query == "exit" {
			break
		}

		results := processQuery(query, index)
		if results == nil {
			fmt.Println("Results found: 0")
		} else {
			fmt.Printf("Results found: %d\n", len(results))
			fmt.Println(results)
		}
	}
}

// loadInvertedIndex загружает инвертированный индекс из JSON файла
func loadInvertedIndex(filename string) (*models.InvertedIndex, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rawIndex map[string][]int
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&rawIndex); err != nil {
		return nil, err
	}

	return models.NewInvertedIndex(rawIndex), nil
}

// processQuery обрабатывает запрос
func processQuery(query string, index *models.InvertedIndex) []int {
	tokens := tokenizeQuery(query)

	if err := validateQuery(tokens); err != nil {
		fmt.Println("Error: ", err)
		return nil
	}

	postfix := infixToPostfix(tokens)
	return evaluatePostfix(postfix, index)
}

// validateQuery проверяет корректность запроса
func validateQuery(tokens []string) error {
	// Проверка на пустой запрос или запрос только из оператора
	if len(tokens) == 0 || (len(tokens) == 1 && isOperator(tokens[0])) {
		return fmt.Errorf("wrong query")
	}

	// Проверка на оператор в конце запроса
	if isOperator(tokens[len(tokens)-1]) {
		return fmt.Errorf("wrong query")
	}

	// Проверка баланса скобок и последовательности операторов
	stack := make([]string, 0)
	for i := range tokens {
		switch tokens[i] {
		case "(":
			stack = append(stack, tokens[i])
		case ")":
			if len(stack) == 0 {
				return fmt.Errorf("wrong query")
			}
			stack = stack[:len(stack)-1]
		default:
			// Проверка последовательности операторов
			if isOperator(tokens[i]) && i+1 < len(tokens) {
				next := tokens[i+1]
				// Разрешаем NOT перед любым токеном, кроме закрывающей скобки, для остальных операторов запрещаем
				if (tokens[i] != "NOT" && isOperator(next)) || next == ")" {
					return fmt.Errorf("wrong query")
				}
			}
		}
	}

	if len(stack) > 0 {
		return fmt.Errorf("wrong query")
	}

	return nil
}

// isOperator проверяет, является ли токен оператором
func isOperator(token string) bool {
	return token == "AND" || token == "OR" || token == "NOT"
}

// tokenizeQuery разбивает запрос на токены
func tokenizeQuery(query string) []string {
	var tokens []string
	var currentToken strings.Builder

	for _, char := range query {
		switch {
		case char == '(' || char == ')':
			if currentToken.Len() > 0 {
				token := currentToken.String()

				switch strings.ToUpper(token) {
				case "AND", "OR", "NOT":
					tokens = append(tokens, strings.ToUpper(token))
				default:
					tokens = append(tokens, lemmatizer.Lemma(strings.ToLower(token)))
				}

				currentToken.Reset()
			}

			tokens = append(tokens, string(char))
		case char == ' ':
			if currentToken.Len() > 0 {
				token := currentToken.String()

				switch strings.ToUpper(token) {
				case "AND", "OR", "NOT":
					tokens = append(tokens, strings.ToUpper(token))
				default:
					tokens = append(tokens, lemmatizer.Lemma(strings.ToLower(token)))
				}

				currentToken.Reset()
			}
		default:
			currentToken.WriteRune(char)
		}
	}

	if currentToken.Len() > 0 {
		token := currentToken.String()

		switch strings.ToUpper(token) {
		case "AND", "OR", "NOT":
			tokens = append(tokens, strings.ToUpper(token))
		default:
			tokens = append(tokens, lemmatizer.Lemma(strings.ToLower(token)))
		}
	}

	return tokens
}

// infixToPostfix конвертирует инфиксную нотацию в постфиксную
func infixToPostfix(tokens []string) []string {
	var postfix []string
	var stack []string

	// Приоритеты операторов
	precedence := map[string]int{
		"NOT": 3,
		"AND": 2,
		"OR":  1,
	}

	for _, token := range tokens {
		switch token {
		case "(":
			stack = append(stack, token)
		case ")":
			// Извлекаем операторы из стека до открывающей скобки
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				postfix = append(postfix, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = stack[:len(stack)-1] // Удаление открывающей скобки
		case "NOT", "AND", "OR":
			// Извлекаем из стека все операторы с большим или равным приоритетом, чтобы они выполнились раньше
			for len(stack) > 0 && stack[len(stack)-1] != "(" &&
				precedence[stack[len(stack)-1]] >= precedence[token] {
				postfix = append(postfix, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, token)
		default:
			postfix = append(postfix, token)
		}
	}

	// Извлечение оставшихся операторов из стека
	for len(stack) > 0 {
		postfix = append(postfix, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return postfix
}

// evaluatePostfix вычисляет постфиксное выражение
func evaluatePostfix(postfix []string, index *models.InvertedIndex) []int {
	var stack [][]int

	// Получаем все документы из индекса
	allPages := make(map[int]bool)
	for _, postings := range index.GetIndex() {
		for _, pageID := range postings {
			allPages[pageID] = true
		}
	}
	var allPagesSlice []int
	for pageID := range allPages {
		allPagesSlice = append(allPagesSlice, pageID)
	}
	sort.Ints(allPagesSlice)

	for _, token := range postfix {
		switch token {
		case "AND":
			// Достаем два последних элемента из стека, выполняем пересечение и кладем результат обратно в стек
			if len(stack) < 2 {
				return nil
			}

			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			sort.Ints(left)
			sort.Ints(right)

			stack = append(stack, intersect(left, right))
		case "OR":
			// Достаем два последних элемента из стека, выполняем объединение и кладем результат обратно в стек
			if len(stack) < 2 {
				return nil
			}
			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			sort.Ints(left)
			sort.Ints(right)

			stack = append(stack, union(left, right))
		case "NOT":
			// NOT в начале выражения, тогда нам нужны все страницы
			if len(stack) < 1 {
				stack = append(stack, allPagesSlice)
				continue
			}

			operand := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			// Создаем множество страниц для исключения
			exclude := make(map[int]bool)
			for _, pageID := range operand {
				exclude[pageID] = true
			}

			// Записываем в результат все страницы, которые не входят в множество исключения
			var result []int
			for _, pageID := range allPagesSlice {
				if !exclude[pageID] {
					result = append(result, pageID)
				}
			}

			stack = append(stack, result)
		default:
			// Добавляем отсортированный массив страниц в стек
			if ids, exists := index.GetIndex()[token]; exists {
				sorted := make([]int, len(ids))
				copy(sorted, ids)
				sort.Ints(sorted)
				stack = append(stack, sorted)
			} else {
				stack = append(stack, []int{})
			}
		}
	}

	if len(stack) != 1 {
		return nil
	}

	return stack[0]
}

// intersect выполняет операцию пересечения двух отсортированных массивов
func intersect(a, b []int) []int {
	var result []int
	i, j := 0, 0

	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			result = append(result, a[i])
			i++
			j++
		} else if a[i] < b[j] {
			i++
		} else {
			j++
		}
	}

	return result
}

// union выполняет операцию объединения двух отсортированных массивов
func union(a, b []int) []int {
	var result []int
	i, j := 0, 0

	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			result = append(result, a[i])
			i++
			j++
		} else if a[i] < b[j] {
			result = append(result, a[i])
			i++
		} else {
			result = append(result, b[j])
			j++
		}
	}

	result = append(result, a[i:]...)
	result = append(result, b[j:]...)

	return result
}
