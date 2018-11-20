package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"time"
)

type ChannelState struct {
	Id     uint
	Offset uint
	Filter string
}

const ConfigPath = "prod.config"
var dbApi DBApi
var templater Templater
var upgrader = websocket.Upgrader{}

func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	fmt.Fprintf(w, `<html><head></head><body><script>window.location.replace("%v")</script></body></html>`, url)
}

func UpdateChannelContent(channelId uint) {
	err := dbApi.UpdateChannelContent(channelId)
	if err != nil {
		log.Println("updating channel error: " + err.Error())
		err = dbApi.MarkChannelAsBroken(channelId)
		log.Printf("mark channel %v as broken", channelId)
		if err != nil {
			log.Println("marking channel as broken error: " + err.Error())
		}
		log.Println(fmt.Sprintf("channel %v marked as broken due to getting content error", channelId))
	}
}

func IndexHandler(writer http.ResponseWriter, request *http.Request) {
	tmpl := templater.GetTemplate("index")
	tmpl.Execute(writer, struct{ Channels []Channel }{Channels: dbApi.ListChannels()})
}

func NewChannelPageHandler(writer http.ResponseWriter, request *http.Request) {
	tmpl := templater.GetTemplate("newchannel")
	tmpl.Execute(writer, struct{ Channels []Channel }{Channels: dbApi.ListChannels()})
}

func AddChannelHandler(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()

	channelName, ok := request.Form["channel_name"]
	if !ok {
		Redirect(writer, request, "/")
		return
	}

	channelSource, ok := request.Form["channel_source"]
	if !ok {
		Redirect(writer, request, "/")
		return
	}

	itemPattern, ok := request.Form["item_pattern"]
	if !ok {
		Redirect(writer, request, "/")
		return
	}

	linkPattern, ok := request.Form["link_pattern"]
	if !ok {
		Redirect(writer, request, "/")
		return
	}

	titlePattern, ok := request.Form["title_pattern"]
	if !ok {
		Redirect(writer, request, "/")
		return
	}

	descriptionPattern, ok := request.Form["description_pattern"]
	if !ok {
		Redirect(writer, request, "/")
		return
	}
	channel, err := dbApi.CreateChannel(channelName[0], channelSource[0], itemPattern[0], linkPattern[0], titlePattern[0], descriptionPattern[0])
	if err != nil {
		log.Println("Creating channel error: " + err.Error())
	}
	go UpdateChannelContent(channel.ID)
	Redirect(writer, request, "/")
}

func ViewChannelHandlerPage(writer http.ResponseWriter, request *http.Request) {
	tmpl := templater.GetTemplate("viewchannel")
	tmpl.Execute(writer, struct{ Channels []Channel }{Channels: dbApi.ListChannels()})
}

func GetChannelContent(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, data, err := c.ReadMessage()
		if err != nil {
			log.Println("reading error:", err)
			break
		}
		var channelState ChannelState
		err = json.Unmarshal(data, &channelState)

		if err != nil {
			log.Println("unmarshalling channel state error:", err)
			break
		}

		posts := dbApi.GetChannelContentWithLimit(channelState.Id, channelState.Offset, 5, channelState.Filter)
		rawPosts, err := json.Marshal(posts)

		if err != nil {
			log.Println("marshalling posts error:", err)
			break
		}

		err = c.WriteMessage(mt, rawPosts)
		if err != nil {
			log.Println("writing error:", err)
			break
		}
	}
}

func DeleteChannelHandler(writer http.ResponseWriter, request *http.Request) {
	strChannelId := request.URL.Path[len("/deletechannel/"):]
	channelId, err := strconv.ParseUint(strChannelId, 10, 32)
	if err != nil {
		log.Println("channel deleting error, bad channel id: " + err.Error())
		return
	}
	err = dbApi.DeleteChannel(uint(channelId))
	if err != nil {
		log.Println("channel deleting error: " + err.Error())
	}
	Redirect(writer, request, "/")
}

func StartServer(configPath string) error {
	config, err := ParseConfig(configPath)
	if err != nil {
		return err
	}
	dbApi.Init(&config.PostgresConfig, config.AddExamples)
	templater.Init(config.TemplatesPath)
	defer dbApi.db.Close()
	go RunUpdater(&dbApi, time.Hour*1, time.Second*3)

	staticDir := fmt.Sprintf("/%v/", config.StaticPath)
	http.Handle(staticDir, http.StripPrefix(staticDir, http.FileServer(http.Dir(config.StaticPath))))

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/newchannel", NewChannelPageHandler)
	http.HandleFunc("/addchannel", AddChannelHandler)
	http.HandleFunc("/deletechannel/", DeleteChannelHandler)
	http.HandleFunc("/channels/", ViewChannelHandlerPage)
	http.HandleFunc("/ws", GetChannelContent)
	http.HandleFunc("/favicon.ico", func(writer http.ResponseWriter, request *http.Request) {})
	log.Println("start server")
	return http.ListenAndServe(fmt.Sprintf("%s:%d", config.Host, config.Port), nil)
}

func main() {
	panic(StartServer(ConfigPath))
}
