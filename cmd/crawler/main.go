package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	totalPages = 100                                 // Кол-во страниц для выкачки
	outputDir  = "pages"                             // Название директории для выкаченных веб-страниц
	baseURL    = "https://elementy.ru/novosti_nauki" // URL ресурса, с которого берутся страницы
	baseDomain = "https://elementy.ru"
)

func main() {
	// Создаем директорию для сохранения страниц
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("create output directory error: %v", err)
	}

	// Создаем файл index.txt для записи номера страницы и URL
	indexFile, err := os.Create("index.txt")
	if err != nil {
		log.Fatalf("create index.txt error: %v", err)
	}
	defer indexFile.Close()

	writer := bufio.NewWriter(indexFile)
	defer writer.Flush()

	urls := make([]string, 0, totalPages)
	basePageNumber := 0

	for len(urls) < totalPages {
		url := fmt.Sprintf("%s?page=%d", baseURL, basePageNumber)
		resp, err := http.Get(url)
		if err != nil {
			log.Fatalf("get page error: %v", err)
		}

		if resp.StatusCode != 200 {
			log.Fatalf("server returned error status code: %v", resp.StatusCode)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		resp.Body.Close()

		// Ищем все ссылки <a> и добавляем в urls
		doc.Find("div.clblock.newslist div.img_block32 a.nohover").Each(func(i int, s *goquery.Selection) {
			// Извлекаем значение атрибута href
			if href, exists := s.Attr("href"); exists {
				if !strings.HasSuffix(href, ".js") && !strings.HasSuffix(href, ".css") && len(urls) < totalPages {
					urls = append(urls, baseDomain+href)
				}
			}
		})

		basePageNumber++
		time.Sleep(100 * time.Millisecond)
	}

	for i, pageURL := range urls {
		resp, err := http.Get(pageURL)
		if err != nil {
			log.Fatalf("get page error: %v", err)
		}

		if resp.StatusCode != 200 {
			log.Fatalf("server returned error status code: %v", resp.StatusCode)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		doc.Find("noscript").Each(func(i int, s *goquery.Selection) {
			s.ReplaceWithHtml(s.Text())
		})

		// Удаляем теги <script> и <link rel='stylesheet'>
		doc.Find("script, link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
			s.Remove()
		})

		cleanedHtml, err := doc.Html()
		if err != nil {
			log.Fatalf("get cleaned html error: %v", err)
		}

		filename := fmt.Sprintf("%s/page_%d.html", outputDir, i+1)

		// Сохраняем страницу
		err = os.WriteFile(filename, []byte(cleanedHtml), 0755)
		if err != nil {
			log.Fatalf("write file error: %v", err)
		}

		// Пишем в index.txt
		_, err = fmt.Fprintf(writer, "%d %s\n", i+1, pageURL)
		if err != nil {
			log.Printf("write to index.txt error: %v", err)
		}

		log.Printf("Saved page: %s", pageURL)

		time.Sleep(100 * time.Millisecond)
	}
}
