package main

import (
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
)

//var itemPattern = regexp.MustCompile("(?s)<article\\sclass=\"post\\spost_preview\">(.*?)</article>")

func DownloadContent(source string) ([]byte, error) {
	result, err := http.Get(source)
	if err != nil {
		return nil, errors.New("downloading content error: " + err.Error())
	}
	defer result.Body.Close()
	content, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return nil, errors.New("downloading content error: " + err.Error())
	}
	return content, nil
}

func getContentByRegexp(name string, value []byte, regexp *regexp.Regexp) (*string, error) {
	titleIndexes := regexp.FindAllStringSubmatchIndex(string(value), -1)
	if len(titleIndexes) != 1 {
		return nil, errors.New(fmt.Sprintf("empty or multiple %v by regexp", name))
	}
	titleIndex := titleIndexes[0]
	if len(titleIndex) < 4 {
		return nil, errors.New(fmt.Sprintf("empty or multiple %v by regexp", name))
	}
	result := string(value[titleIndex[2]:titleIndex[3]])
	return &result, nil
}

func ParseContent(rule *CompiledRule, content []byte) ([]Post, error) {
	var posts []Post
	itemPattern := rule.ItemPattern

	indexes := itemPattern.FindAllSubmatchIndex(content, -1)

	if len(indexes) == 0 {
		return nil, errors.New("can not find item by regexp, len(indexes) = 0")
	}
	for _, index := range indexes {
		if len(index) < 4 {
			log.Println(len(content))
			log.Println(index)
			return nil, errors.New(fmt.Sprintf("can not find item by regexp, len(index) = %d", len(index)))
		}
		itemLowerIndex := index[2]
		itemUpperIndex := index[3]
		itemContent := content[itemLowerIndex:itemUpperIndex]

		title, err := getContentByRegexp("title", itemContent, &rule.TitlePattern)
		if err != nil {
			return nil, errors.New(err.Error())
		}

		link, err := getContentByRegexp("link", itemContent, &rule.LinkPattern)
		if err != nil {
			return nil, errors.New(err.Error())
		}

		description, err := getContentByRegexp("description", itemContent, &rule.DescriptionPattern)
		if err != nil {
			return nil, errors.New(err.Error())
		}

		posts = append(posts, Post{Title: string(*title), Link: string(*link), Description: string(*description)})
	}
	return posts, nil
}

func GetContent(source string, rule *CompiledRule) ([]Post, error) {
	content, err := DownloadContent(source)
	if err != nil {
		return nil, errors.New("getting content error: " + err.Error())
	}

	content = []byte(html.UnescapeString(string(content)))
	posts, err := ParseContent(rule, content)
	if err != nil {
		return nil, errors.New("getting content error: " + err.Error())
	}
	return posts, nil
}
