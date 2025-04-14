package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bzick/tokenizer"
)

const (
	pagesDir  = "pages"
	tokensDir = "tokens"
	lemmasDir = "lemmas"

	tokensTfIdfDir = "tokens_tf_idf"
	lemmasTfIdfDir = "lemmas_tf_idf"
)

func main() {
	// Создание директории для TF-IDF токенов
	if err := os.MkdirAll(tokensTfIdfDir, 0755); err != nil {
		log.Fatalf("create tokens tf-idf directory error: %v", err)
	}

	// Создание директории для TF-IDF лемм
	if err := os.MkdirAll(lemmasTfIdfDir, 0755); err != nil {
		log.Fatalf("create lemmas tf-idf directory error: %v", err)
	}

	parser := tokenizer.New()

	allTokens, err := getAllTokens()
	if err != nil {
		log.Fatal(err)
	}

	pages, err := os.ReadDir(pagesDir)
	if err != nil {
		log.Fatal(err)
	}

	// Проходимся по каждой странице
	for _, page := range pages {
		pageFile, err := os.Open(pagesDir + "/" + page.Name())
		if err != nil {
			log.Fatal(err)
		}

		doc, err := goquery.NewDocumentFromReader(pageFile)
		if err != nil {
			log.Fatal(err)
		}

		words := make([]string, 0)

		// Достаем контент страницы и записываем все слова в массив words
		doc.Find("div.body div.mblock div.itemblock div.memo").Each(func(i int, s *goquery.Selection) {
			wordsStream := parser.ParseString(s.Text())
			for wordsStream.IsValid() {
				word := strings.ToLower(wordsStream.CurrentToken().ValueString())
				words = append(words, word)
				wordsStream.GoNext()
			}
			wordsStream.Close()
		})

		var pageNum int
		_, err = fmt.Sscanf(page.Name(), "page_%d.html", &pageNum)
		if err != nil {
			log.Fatal(err)
		}

		// Создание файла с TF-IDF токенов страницы
		tokensTfIdfFile, err := os.Create(fmt.Sprintf("%s/tokens_tf_idf_%d.txt", tokensTfIdfDir, pageNum))
		if err != nil {
			log.Fatal(err)
		}

		tokensTfIdfWriter := bufio.NewWriter(tokensTfIdfFile)

		// Вычисление TF-IDF и запись в файл
		for _, token := range allTokens[pageNum] {
			tf := float64(countTokenOccurrencesInPage(words, token)) / float64(len(words))
			idf := math.Log(float64(len(pages)) / float64(countPagesWithToken(allTokens, token)))

			_, err = fmt.Fprintf(tokensTfIdfWriter, "%s %f %f\n", token, idf, tf*idf)
			if err != nil {
				log.Printf("write to tokens tf-idf file error: %v", err)
				continue
			}
		}

		// Создание файла с TF-IDF лемм страницы
		lemmasTfIdfFile, err := os.Create(fmt.Sprintf("%s/lemmas_tf_idf_%d.txt", lemmasTfIdfDir, pageNum))
		if err != nil {
			log.Fatal(err)
		}

		lemmasTfIdfWriter := bufio.NewWriter(lemmasTfIdfFile)

		lemmasFile, err := os.Open(fmt.Sprintf("%s/lemmas_%d.txt", lemmasDir, pageNum))
		if err != nil {
			log.Fatal(err)
		}

		lemmasScanner := bufio.NewScanner(lemmasFile)

		// Вычисление TF-IDF лемм и запись в файл
		for lemmasScanner.Scan() {
			line := lemmasScanner.Text()

			parts := strings.Split(line, ": ")

			lemma := parts[0]
			tokens := strings.Split(parts[1], " ")

			var lemmaTf float64

			for _, token := range tokens {
				lemmaTf += float64(countTokenOccurrencesInPage(words, token)) / float64(len(words))
			}

			lemmaIdf := math.Log(float64(len(pages)) / float64(countPagesWithLemma(allTokens, tokens)))

			_, err = fmt.Fprintf(lemmasTfIdfWriter, "%s %f %f\n", lemma, lemmaIdf, lemmaTf*lemmaIdf)
			if err != nil {
				log.Printf("write to lemmas tf-idf file error: %v", err)
				continue
			}
		}
		if err = lemmasScanner.Err(); err != nil {
			log.Fatal(err)
		}

		tokensTfIdfWriter.Flush()
		tokensTfIdfFile.Close()
		lemmasTfIdfWriter.Flush()
		lemmasTfIdfFile.Close()
		lemmasFile.Close()
		pageFile.Close()
	}
}

// getAllTokens возвращает мапу, где ключ - номер страницы, значение - массив токенов
func getAllTokens() (map[int][]string, error) {
	items, err := os.ReadDir(tokensDir)
	if err != nil {
		return nil, err
	}

	tokens := make(map[int][]string)

	for _, item := range items {
		var pageNum int
		_, err := fmt.Sscanf(item.Name(), "tokens_%d.txt", &pageNum)
		if err != nil {
			return nil, err
		}

		if _, ok := tokens[pageNum]; !ok {
			tokens[pageNum] = make([]string, 0)
		}

		file, err := os.Open(tokensDir + "/" + item.Name())
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := scanner.Text()

			token := strings.TrimSpace(line)
			if token == "" {
				continue
			}

			tokens[pageNum] = append(tokens[pageNum], token)
		}
		if err = scanner.Err(); err != nil {
			return nil, err
		}

		file.Close()
	}

	return tokens, nil
}

// countTokenOccurrencesInPage возвращает количество появлений токена на странице
func countTokenOccurrencesInPage(pageWords []string, token string) int {
	count := 0
	for _, word := range pageWords {
		if word == token {
			count++
		}
	}

	return count
}

// countPagesWithToken возвращает количество страниц, в которых встречается токен
func countPagesWithToken(allTokens map[int][]string, token string) int {
	count := 0
	for _, tokens := range allTokens {
		for _, t := range tokens {
			if t == token {
				count++
				break
			}
		}
	}

	return count
}

// countPagesWithLemma возвращает количество страниц, в котрых встречается лемма
func countPagesWithLemma(allTokens map[int][]string, lemmaTokens []string) int {
	count := 0
	for _, tokens := range allTokens {
		for _, t := range tokens {
			tokenFound := false
			for _, lemmaToken := range lemmaTokens {
				if t == lemmaToken {
					count++
					tokenFound = true
				}
			}
			if tokenFound {
				break
			}
		}
	}

	return count
}
