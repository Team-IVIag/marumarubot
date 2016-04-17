package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	QueryFormat   = "http://marumaru.in/?r=home&mod=search&keyword=%v&x=0&y=0"
	ResultRegex   = ".*\\/b\\/manga/([0-9]*)"
	MaruPrefix    = "http://marumaru.in"
	ArchiveFormat = "http://marumaru.in/b/manga/%d"
	ShenRegex     = "http:\\/\\/www\\.shencomics\\.com\\/archives\\/[0-9]*"
	ShenPrefix    = "http://www.shencomics.com/archives/"
)

type KeySortedMap struct{
	val map[string]string
	key []string
}

func (m *KeySortedMap) Set(key string, value string){
	m.key = append(m.key, key)
	m.val[key] = value
}

func (m *KeySortedMap) Get(key string) string{
	return m.val[key]
}

func query(keyword string) (links, names []string, indexes []int, err error) {
	doc, err := goquery.NewDocument(fmt.Sprintf(QueryFormat, url.QueryEscape(keyword)))
	if err != nil {
		return
	}

	doc.Children().Find(".subject").Each(func(i int, s *goquery.Selection) {
		l, ok := s.Attr("href")

		if ok {
			if match, _ := regexp.MatchString(ResultRegex, l); match {
				links = append(links, l)

				i := strings.LastIndex(l, "/")
				index, err := strconv.Atoi(l[i+1:])
				indexes = append(indexes, index)
				if err != nil {
					return
				}

				s.Children().Find(".sbjbox").Each(func(i int, s1 *goquery.Selection) {
					html, _ := s1.Html()
					names = append(names, html[4:len(html)-5])
				})
			}
		}
	})
	return
}

func getList(id, page int) (list KeySortedMap, err error) {
	doc, err := goquery.NewDocument(fmt.Sprintf(ArchiveFormat, id))
	if err != nil {
		return
	}

	list = KeySortedMap{
		val: make(map[string]string),
	}

	doc.Children().Find("a").Each(func(i int, s *goquery.Selection) {
		l, ok := s.Attr("href")

		if ok {
			if match, _ := regexp.MatchString(ShenRegex, l); match {
				html, _ := s.Html()
				html = strings.Replace(strings.Replace(html, "(", "\\(", -1), ")", "\\)", -1)

				name := ""
				open := false
				for _, v := range []rune(html) {
					if v == '<' {
						open = true
					}else if v == '>' {
						open = false
						continue
					}

					if !open {
						name += string(v)
					}
				}
				list.Set(l[strings.LastIndex(l, "/")+1:], name)
			}
		}
	})
	return
}
