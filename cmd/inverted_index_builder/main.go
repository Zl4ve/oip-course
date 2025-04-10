package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"oip-course/internal/models"
	"os"
	"strings"
)

const (
	lemmasDir = "lemmas"
)

// processFile читает файл, извлекает леммы и добавляет номер страницы в инвертированный индекс
func processFile(fileName string, ii *models.InvertedIndex) {
	// Получаем номер страницы
	var pageNum int
	_, err := fmt.Sscanf(fileName, "lemmas_%d.txt", &pageNum)
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Open(lemmasDir + "/" + fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Построчно обрабатываем файл, достаем лемму и записываем в индекс
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}
		lemma := strings.TrimSpace(parts[0])
		if lemma == "" {
			continue
		}

		ii.Add(lemma, pageNum)
	}
	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	ii := models.NewInvertedIndex(make(map[string][]int))

	// Получаем список файлов лемм
	items, err := os.ReadDir(lemmasDir)
	if err != nil {
		log.Fatal(err)
	}

	// Запускаем обработку лемм для каждого файла
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		name := item.Name()
		if !strings.HasPrefix(name, "lemmas_") || !strings.HasSuffix(name, ".txt") {
			continue
		}

		processFile(name, ii)
	}

	// Преобразуем индекс в формат JSON
	jsonData, err := json.MarshalIndent(ii.GetIndex(), "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	// Записываем JSON в файл
	err = os.WriteFile("inverted_index.json", jsonData, 0755)
	if err != nil {
		log.Fatal(err)
	}
}
