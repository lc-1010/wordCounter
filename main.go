package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type wordFrequencies struct {
	Key   string
	Value int
}

func main() {
	url := "https://github.com/rust-lang/book/tree/main/src"
	rawUrl := "https://raw.githubusercontent.com/rust-lang/book/main/src/"

	urls := urls{
		Url:    url,
		RawUrl: rawUrl,
	}
	fileName := "word_list.txt"
	fileExtension := "ch"
	res := urls.readRepo(fileExtension)
	writeFile(res, fileName)
}

func writeFile(res []wordFrequencies, fileName string) {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	for _, item := range res {
		if len(item.Key) > 15 { //catastrophic
			continue
		}
		_, err := file.Write([]byte(item.Key + " "))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getContent(url string) string {
	respone, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer respone.Body.Close()
	body, err := io.ReadAll(respone.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(body)
}

func getWordList(body string) []string {
	content := cleanText(string(body))
	wordList := strings.Fields(content)
	return wordList

}

func sortMapByValue(word *sync.Map) []wordFrequencies {
	var sortedWords []wordFrequencies

	word.Range(func(key, value any) bool {
		sortedWords = append(sortedWords, wordFrequencies{
			Key:   key.(string),
			Value: value.(int),
		})
		return true
	})

	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[i].Value > sortedWords[j].Value
	})
	return sortedWords
}

func cleanText(text string) string {
	// 去除标点符号
	punctuationRegex := regexp.MustCompile(`[[:punct:]]`)
	cleanedText := punctuationRegex.ReplaceAllString(text, "")

	// 转换为小写字母
	cleanedText = strings.ToLower(cleanedText)

	return cleanedText
}

func ExtractEmbeddedUrlsFromGithub(url string, rawUrl string, fileExtension string) []string {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var res []string
	document.Find("script[type='application/json']").Each(func(i int, s *goquery.Selection) {
		content := s.Text()
		var raw json.RawMessage

		err := json.Unmarshal([]byte(content), &raw)
		if err != nil {
			log.Fatal(err)
		}
		var data map[string]any
		err = json.Unmarshal(raw, &data)
		if err != nil {
			log.Fatal(err)
		}
		payload, ok := data["payload"].(map[string]any)

		if ok {
			tree, ok := payload["tree"].(map[string]any)
			if ok {
				if tree["items"] != nil {
					item := tree["items"].([]any)
					for _, k := range item {
						ff := k.(map[string]any)["name"].(string)
						if strings.HasPrefix(ff, fileExtension) {
							res = append(res, rawUrl+ff)
						}
					}
				}
			}
		}
	})
	return res
}

type urls struct {
	Url    string
	RawUrl string
}

func (u urls) readRepo(fileExtension string) []wordFrequencies {
	done := make(chan struct{})
	var wg sync.WaitGroup
	var wordMap sync.Map
	wg.Add(1)
	go func(u string, rawUrl string, w *sync.Map) {
		defer wg.Done()
		processPage(u, rawUrl, w, fileExtension)
	}(u.Url, u.RawUrl, &wordMap)

	wg.Wait()
	close(done)

	strotedWords := sortMapByValue(&wordMap)
	return strotedWords
}

func processPage(u string, r string, wordMap *sync.Map, fileExtension string) {
	list := ExtractEmbeddedUrlsFromGithub(u, r, fileExtension)
	wordChan := make(chan []string, len(list))
	done := make(chan struct{})
	var wg sync.WaitGroup

	for _, item := range list {
		wg.Add(1)
		go func(itme string) {
			defer wg.Done()
			content := getContent(itme)
			wordList := getWordList(content)
			wordChan <- wordList
		}(item)

	}
	go func() {
		wg.Wait()
		close(done)
	}()
	countWordFrequencies(wordChan, done, wordMap)
}

func countWordFrequencies(w chan []string, d chan struct{}, wordMap *sync.Map) *sync.Map {
	for {
		select {
		case word := <-w:
			for _, w := range word {
				c, res := wordMap.LoadOrStore(w, 1)
				if res {
					wordMap.Store(w, c.(int)+1)
				}
			}
		case <-d:
			return wordMap
		}
	}
}
