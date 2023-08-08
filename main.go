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
type wordMap struct {
	word map[string]int
	sync.Mutex
}

func main() {
	url := "https://github.com/rust-lang/book/tree/main/src"
	rawUrl := "https://raw.githubusercontent.com/rust-lang/book/main/src/"
	lists := ExtractEmbeddedUrlsFromGithub(url, rawUrl)
	word := make(chan []string, len(lists))
	freq := make(chan map[string]int, 1)
	done := make(chan int, 1)

	for _, item := range lists {
		go getContent(item, word, done)
	}
	countWordFrequencies(word, freq, done, len(lists))

	wordMapList := <-freq
	//fmt.Println("wordMapList", wordMapList)
	res := sortMapByValue(wordMapList)
	file, err := os.OpenFile("word_list.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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

func getContent(url string, w chan []string, d chan int) {
	body := getBody(url)
	readMd(body, w, d)
}

func getBody(url string) string {
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

func readMd(body string, w chan []string, d chan int) {
	content := cleanText(string(body))
	wordList := strings.Fields(content)
	w <- wordList

}

func sortMapByValue(word map[string]int) []wordFrequencies {
	var sortedWords []wordFrequencies
	for w, frequency := range word {
		sortedWords = append(sortedWords, wordFrequencies{
			Key:   w,
			Value: frequency,
		})
	}
	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[i].Value > sortedWords[j].Value
	})
	return sortedWords
}

func countWordFrequencies(w chan []string, freq chan map[string]int, cnt chan int, pageCnt int) {
	i := 0
	res := wordMap{
		word: map[string]int{},
	}
	for {
		select {
		case words := <-w:
			for _, word := range words {

				res.Lock()
				res.word[word]++
				res.Unlock()

			}
			i++
			if i == pageCnt {
				cnt <- len(res.word)
			}
		case <-cnt:
			freq <- res.word
			return
		}
	}
}

func cleanText(text string) string {
	// 去除标点符号
	punctuationRegex := regexp.MustCompile(`[[:punct:]]`)
	cleanedText := punctuationRegex.ReplaceAllString(text, "")

	// 转换为小写字母
	cleanedText = strings.ToLower(cleanedText)

	return cleanedText
}

func ExtractEmbeddedUrlsFromGithub(url string, rawUrl string) []string {
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
						if strings.HasPrefix(ff, "ch") {
							res = append(res, rawUrl+ff)
						}
					}
				}
			}
		}
	})
	return res
}
