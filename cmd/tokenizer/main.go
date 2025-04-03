package main

import (
	"bufio"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/aaaton/golem/v4"
	"github.com/aaaton/golem/v4/dicts/ru"
	"github.com/bbalet/stopwords"
	"github.com/bzick/tokenizer"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	pagesDir  = "pages"
	tokensDir = "tokens"
	lemmasDir = "lemmas"
)

// Регулярное выражение, проверяющее, что запись состоит из русских букв
var russianWordRegexp = regexp.MustCompile("^[А-ЯЁа-яё]+$")

func main() {

	// Создание директории для токенов
	if err := os.MkdirAll(tokensDir, 0755); err != nil {
		log.Fatalf("create tokens directory error: %v", err)
	}

	// Создание директории для лемм
	if err := os.MkdirAll(lemmasDir, 0755); err != nil {
		log.Fatalf("create lemmas directory error: %v", err)
	}

	parser := tokenizer.New()

	lemmatizer, err := golem.New(ru.New())
	if err != nil {
		log.Fatal(err)
	}

	// Читаем все файлы в директории pages
	items, err := os.ReadDir(pagesDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range items {
		pageFile, err := os.Open(pagesDir + "/" + item.Name())
		if err != nil {
			log.Fatal(err)
		}

		doc, err := goquery.NewDocumentFromReader(pageFile)
		if err != nil {
			log.Fatal(err)
		}

		tokens := make([]string, 0)

		// Достаем контент страницы и заполняем массив tokens русскими словами, игнорируя стоп-слова
		doc.Find("div.body div.mblock div.itemblock div.memo").Each(func(i int, s *goquery.Selection) {
			tokensStream := parser.ParseString(s.Text())
			for tokensStream.IsValid() {
				token := strings.ToLower(tokensStream.CurrentToken().ValueString())
				if isRussianWord(token) && strings.TrimSpace(stopwords.CleanString(token, "ru", false)) != "" && utf8.RuneCountInString(token) > 2 {
					tokens = append(tokens, token)
				}

				tokensStream.GoNext()
			}

			tokensStream.Close()
		})

		var pageNum int
		_, err = fmt.Sscanf(item.Name(), "page_%d.html", &pageNum)
		if err != nil {
			log.Fatal(err)
		}

		// Создание файла с токенами
		tokensFile, err := os.Create(fmt.Sprintf("%s/tokens_%d.txt", tokensDir, pageNum))
		if err != nil {
			log.Fatal(err)
		}

		tokensWriter := bufio.NewWriter(tokensFile)

		// Заполняем мапу, где ключ - лемма, значение - массив токенов
		lemmasMap := make(map[string][]string)
		for _, token := range tokens {
			// Записываем токен в файл
			_, err = fmt.Fprintf(tokensWriter, "%s\n", token)
			if err != nil {
				log.Printf("write to tokens file error: %v", err)
			}

			lemma := lemmatizer.Lemma(token)
			if _, ok := lemmasMap[lemma]; !ok {
				lemmasMap[lemma] = make([]string, 0)
			}

			if !slices.Contains(lemmasMap[lemma], token) {
				lemmasMap[lemma] = append(lemmasMap[lemma], token)
			}
		}

		// Создание файла с леммами
		lemmasFile, err := os.Create(fmt.Sprintf("%s/lemmas_%d.txt", lemmasDir, pageNum))
		if err != nil {
			log.Fatal(err)
		}

		lemmasWriter := bufio.NewWriter(lemmasFile)

		// Запись лемм и токенов в файл
		for lemma, tokens := range lemmasMap {
			lemmaStr := lemma + ":"
			for _, token := range tokens {
				lemmaStr = lemmaStr + " " + token
			}

			_, err = fmt.Fprintf(lemmasWriter, "%s\n", lemmaStr)
			if err != nil {
				log.Printf("write to lemmas file error: %v", err)
			}
		}

		pageFile.Close()
		tokensWriter.Flush()
		lemmasWriter.Flush()
		tokensFile.Close()
		lemmasFile.Close()
	}

}

// Проверка, что слово слово состоит из русских букв
func isRussianWord(word string) bool {
	return russianWordRegexp.MatchString(word)
}
