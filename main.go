package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type wordFrequencies struct {
	Key   string
	Value int
}

func main() {
	url := "https://github.com/rust-lang/book/tree/main/src"
	rawUrl := "https://raw.githubusercontent.com/rust-lang/book/main/src/"
	list := ExtractEmbeddedUrlsFromGithub(url, rawUrl)
	word := make(chan string, 1)
	freq := make(chan map[string]int)
	for _, l := range list {
		go readMd(l, word, freq)
	}
	countWordFrequencies(list, word, freq)

}

func readMd(url string, w chan string, f chan map[string]int) {
	respone, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer respone.Body.Close()
	body, err := io.ReadAll(respone.Body)
	if err != nil {
		log.Fatal(err)
	}
	content := cleanText(string(body))
	wordList := strings.Fields(content)
	//fmt.Println(wordList[10:30])

	countWordFrequencies(wordList, w, f)
	// result := sortMapByValue(res)
	// fmt.Println(len(result))
	// for _, w := range result {
	// 	fmt.Println(w)
	// }

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

func countWordFrequencies(words []string, w chan string, freq chan map[string]int) {
	wordFrequencies := make(map[string]int)
	for _, word := range words {
		wordFrequencies[word]++
	}
	freq <- wordFrequencies
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
