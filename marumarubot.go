package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/telegram-bot-api.v3"
)

var config map[string]interface{}

const (
	TypeQuery = iota
	TypeArchive
)

type Counter struct {
	count map[int]int
	mux   sync.Mutex
}

var counter Counter = Counter{count: make(map[int]int)}

func initConfig() {
	tmp := make(map[string]interface{})
	tmp["token"] = "<your_token_here>"
	tmp["max-count"] = 3

	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		content, err := json.Marshal(tmp)
		if err != nil {
			log.Panic(err)
		}

		ioutil.WriteFile("config.json", content, 0666)

		config = tmp
	} else {
		content, _ := ioutil.ReadFile("config.json")
		json.Unmarshal(content, &config)
	}

	for key, val := range tmp {
		if _, ok := config[key]; !ok {
			config[key] = val
		}
	}
}

func main() {
	initConfig()

	bot, err := tgbotapi.NewBotAPI(config["token"].(string))
	if err != nil {
		log.Panic(err)
	}

	started := getNow()

	log.Printf("Logged in as %#v", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	messages, err := bot.GetUpdatesChan(u)

	for message := range messages {
		log.Printf("[#%v] %v: %#v", message.Message.From.ID, message.Message.From.UserName, message.Message.Text)

		command, args := parseCommand(message.Message.Text)

		if started > message.Message.Date {
			continue
		}

		switch command {
		case "mq":
			if len(args) <= 0 {
				bot.Send(newMessage("Usage: /mq <query...>", message.Message.Chat.ID, message.Message.MessageID))
				break
			}

			keyword := strings.Join(args, " ")

			go func() {
				if counter.count[TypeQuery] >= config["max-count"].(int) {
					bot.Send(newMessage("Server has too many requests at a time. Please try again later.", message.Message.Chat.ID, message.Message.MessageID))
					return
				}
				counter.mux.Lock()
				counter.count[TypeQuery]++
				counter.mux.Unlock()

				defer func() {
					counter.mux.Lock()
					counter.count[TypeQuery]--
					counter.mux.Unlock()
				}()

				links, names, indexes, _ := query(keyword)

				str := ""
				for i := 0; i < len(links); i++ {
					str += "[" + strconv.Itoa(indexes[i]) + "](" + MaruPrefix + links[i] + "): " + names[i] + "\n"
				}
				bot.Send(newMessage(str, message.Message.Chat.ID, message.Message.MessageID))
			}()
			break
		case "mlist":
			if len(args) <= 0 {
				bot.Send(newMessage("Usage: /mlist <id>", message.Message.Chat.ID, message.Message.MessageID))
				break
			}

			go func() {
				if counter.count[TypeQuery] >= config["max-count"].(int) {
					bot.Send(newMessage("Server has too many requests at a time. Please try again later.", message.Message.Chat.ID, message.Message.MessageID))
					return
				}
				counter.mux.Lock()
				counter.count[TypeQuery]++
				counter.mux.Unlock()

				defer func() {
					counter.mux.Lock()
					counter.count[TypeQuery]--
					counter.mux.Unlock()
				}()

				i, _ := strconv.Atoi(args[0])
				list, err := getList(i, 1)

				if err != nil {
					bot.Send(newMessage("Error", message.Message.Chat.ID, message.Message.MessageID))
					return
				}

				str := ""
				for _, id := range list.key {
					name := list.val[id]
					fmt.Println(name, ShenPrefix+id)
					str += fmt.Sprintf("[%v](%v)\n", name, ShenPrefix+id)
				}
				bot.Send(newMessage(str, message.Message.Chat.ID, message.Message.MessageID))
			}()
			break
		case "mget": // TODO
			if len(args) <= 0 {
				bot.Send(newMessage("Usage: /mget <id>", message.Message.Chat.ID, message.Message.MessageID))
				break
			}

			go func() {
				if counter.count[TypeArchive] >= config["max-count"].(int) {
					bot.Send(newMessage("Server has too many requests at a time. Please try again later.", message.Message.Chat.ID, message.Message.MessageID))
					return
				}
				counter.mux.Lock()
				counter.count[TypeArchive]++
				counter.mux.Unlock()

				defer func() {
					counter.mux.Lock()
					counter.count[TypeArchive]--
					counter.mux.Unlock()
				}()

				i, _ := strconv.Atoi(args[0])

				lp := LinkParser{}
				links, _ := lp.Get(i)

				dl := Downloader{archiveId: i, links: links}
				paths, err := dl.Get()

				for _, id := range paths.key {
					path := paths.val[id]
					photo := tgbotapi.NewPhotoUpload(int64(message.Message.From.ID), path)

					log.Println(path)

					_, err = bot.Send(photo)
					if err != nil {
						log.Println(err)
					}
				}
				//				path, err := concatImage(dl.baseFolder, paths)

				if err != nil {
					log.Println(err)
					return // TODO
				}

				if err != nil {
					bot.Send(newMessage("Error", message.Message.Chat.ID, message.Message.MessageID))
					return
				}
			}()
			break
		}
	}
}

func parseCommand(line string) (cmd string, args []string) {
	lines := strings.Split(line, " ")
	if len(lines) <= 0 {
		return
	}

	if len(lines[0]) > 1 && lines[0][0] == '/' {
		cmd = lines[0][1:]
	} else {
		return
	}

	if len(lines) > 1 {
		args = lines[1:]
	}

	index := strings.Index(cmd, "@")
	if index != -1 {
		cmd = cmd[:index-1]
	}

	return
}

func newMessage(message string, sendTo int64, replyTo int) (msg tgbotapi.MessageConfig) {
	msg = tgbotapi.NewMessage(sendTo, message)
	msg.ReplyToMessageID = replyTo
	msg.ParseMode = tgbotapi.ModeMarkdown
	return
}

func getNow() int {
	return int(time.Now().Unix())
}

func concatImage(folder string, paths KeySortedMap) (string, error) {
	var images []image.Image

	width, height := 0, 0
	for _, id := range paths.key {
		path := paths.val[id]
		log.Println(path)

		file, err := os.Open(path)
		if err != nil {
			continue
		}

		if f, err := file.Stat(); !os.IsNotExist(err) {
			name := f.Name()
			log.Println(name)
			if strings.HasSuffix(name, ".jpg") {
				img, _, _ := image.Decode(file)
				images = append(images, img)

				width = max(width, img.Bounds().Max.X-img.Bounds().Min.X)
				height += (img.Bounds().Max.Y - img.Bounds().Min.Y)
			}
		}
	}

	rgba := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{width, height}})

	now := 0
	for _, img := range images {
		w := (img.Bounds().Max.X - img.Bounds().Min.X)
		h := (img.Bounds().Max.Y - img.Bounds().Min.Y)
		draw.Draw(rgba, image.Rectangle{image.Point{0, now}, image.Point{w, now + h}}, img, image.Point{0, 0}, draw.Src)

		now += h
	}

	out, err := os.Create(folder + "result.jpg")
	if err != nil {
		return "", err
	}

	var opt jpeg.Options
	opt.Quality = 80

	jpeg.Encode(out, rgba, &opt)
	out.Close()

	return folder + "result.jpg", nil

}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
