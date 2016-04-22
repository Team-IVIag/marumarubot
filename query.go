package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/organization/cloudflare-bypass"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	UserAgent = "Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US) AppleWebKit/534.7 (KHTML, like Gecko) Chrome/7.0.517.44 Safari/534.7"

	QueryFormat   = "http://marumaru.in/?r=home&mod=search&keyword=%v&x=0&y=0"
	ResultRegex   = ".*\\/b\\/manga/([0-9]*)"
	MaruPrefix    = "http://marumaru.in"
	ArchiveFormat = "http://marumaru.in/b/manga/%d"
	ShenRegex     = "http:\\/\\/www\\.shencomics\\.com\\/archives\\/[0-9]*"
	ShenPrefix    = "http://www.shencomics.com/archives/"
	ImageRegex    = "http://www.shencomics.com/wp-content/uploads/[0-9]{4}/[0-9]{2}/(.*)\\.jpg.*"
)

var imagePtrn = regexp.MustCompile(ImageRegex)

type KeySortedMap struct {
	val map[string]string
	key []string
}

func (m *KeySortedMap) Set(key string, value string) {
	m.key = append(m.key, key)
	m.val[key] = value
}

func (m *KeySortedMap) Get(key string) string {
	return m.val[key]
}

type LinkParser struct {
	Cookie http.Cookie
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

func getList(id int) (list KeySortedMap, err error) {
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
					} else if v == '>' {
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

func (l *LinkParser) Get(id int) (links KeySortedMap, err error) {
	req, err := http.NewRequest("GET", ShenPrefix+strconv.Itoa(id), nil)

	req.Header.Set("User-Agent", UserAgent)
	req.AddCookie(&l.Cookie)
	links = KeySortedMap{val: make(map[string]string)}
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return
	}

	if doc.Find("title").First().Text()[:7] == "You are" {
		l.UpdateCookie(doc)
		return l.Get(id)
	}

	doc.Find("div .entry-content").Find("img").Each(func(i int, s *goquery.Selection) {
		l, ok := s.Attr("data-lazy-src")

		if ok {
			if match := imagePtrn.FindStringSubmatch(l); match != nil {
				links.Set(match[1], l)
			}
		}
	})

	return
}

func (l *LinkParser) UpdateCookie(doc *goquery.Document) {
	s := cfbypass.DecodeScript(doc)
	key := cfbypass.GetCookieKey(s[1])
	val := cfbypass.GetCookieValue(s[0])

	l.Cookie = http.Cookie{}
	l.Cookie.Name = key
	l.Cookie.Value = val
}
