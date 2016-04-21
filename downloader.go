package main

import (
	"io"
	"net/http"
	"os"
	"strconv"
)

type Downloader struct {
	links      KeySortedMap
	baseFolder string
	chats      []int
	archiveId  int
}

func (d *Downloader) Get() (paths KeySortedMap, err error) {
	if _, err := os.Stat(".temp"); os.IsNotExist(err) {
		os.Mkdir(".temp", 0777)
	}

	d.baseFolder = ".temp/" + strconv.Itoa(d.archiveId) + "/"
	if _, err := os.Stat(d.baseFolder); os.IsNotExist(err) {
		os.Mkdir(d.baseFolder, 0777)
	}

	paths = KeySortedMap{val: make(map[string]string)}
	for _, id := range d.links.key {
		link := d.links.val[id]
		out, err := os.Create(d.baseFolder + id + ".jpg")
		if err != nil {
			return paths, err
		}
		defer out.Close()

		resp, err := http.Get(link)
		defer resp.Body.Close()

		if err != nil {
			continue
		}

		_, err = io.Copy(out, resp.Body)

		if err != nil {
			return paths, err
		}

		paths.Set(id, d.baseFolder+id+".jpg")
	}

	return
}
