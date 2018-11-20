package main

import (
	"encoding/json"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func startWebServer() {
	panic(StartServer(TestConfigPath))
}

func TestServerPing(t *testing.T) {
	deferFunc, err := StartPostgres()
	if err != nil {
		panic(err)
	}
	defer deferFunc()
	log.Println("successfully started test db in the docker")

	go startWebServer()

	time.Sleep(1000 * time.Millisecond)

	Convey("Server pinging", t, func() {
		Convey("Server should return 200 OK at start", func() {
			r, _ := http.Get("http://localhost:8080/")
			So(r.StatusCode, ShouldEqual, http.StatusOK)
		})
	})
}

var habrRule = Rule{
	ItemPattern: "(?s)<article\\sclass=\"post\\spost_preview\">(.*?)</article>",
	LinkPattern: "<a\\shref=\"(.*?)\"\\sclass=\"post__title_link\">.*?</a>",
	TitlePattern: "<a\\shref=\".*?\"\\sclass=\"post__title_link\">(.*?)</a>",
	DescriptionPattern: "(?s)<div\\sclass=\"post__text\\spost__text-html\\sjs-mediator-article\">(.*?)</div>\\s\\s\\s\\s\\s\\s\\s\\s\\s\\s<a class=\"btn\\sbtn_x-large\\sbtn_outline_blue\\spost__habracut-btn\"",
}
var compiledHabrRule, _ = CompileRule(&habrRule)

func getExpectedPosts(filename string) ([]Post, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	data = []byte(html.UnescapeString(string(data)))

	var expectedPosts []Post
	err = json.Unmarshal(data, &expectedPosts)
	if err != nil {
		return nil, err
	}
	return expectedPosts, nil
}

func addHabrChannel(api *DBApi, mockedSource string) (*Channel, error) {
	habrChannel, err := dbApi.CreateChannel(
		"Habr",
		mockedSource,
		habrRule.ItemPattern,
		habrRule.LinkPattern,
		habrRule.TitlePattern,
		habrRule.DescriptionPattern,
	)
	if err != nil {
		return nil, errors.New("can not create habr channel: " + err.Error())
	}
	api.db.Preload("Rule").Where("ID = ?", habrChannel.ID).Find(&habrChannel)
	return habrChannel, nil
}

var upRule = Rule{
	ItemPattern: "(?s)<item>(.*?)</item>",
	LinkPattern: "(?s)<link>(.*?)</link>",
	TitlePattern: "<title>(.*?)</title>",
	DescriptionPattern: "(?s)<description>(.*?)</description>",
}

var compiledUpRule, _ = CompileRule(&upRule)

func addUbuntuPlanetChannel(api *DBApi, mockedSource string) (*Channel, error) {
	upChannel, err := dbApi.CreateChannel(
		"Ubuntu Planet",
		mockedSource,
		upRule.ItemPattern,
		upRule.LinkPattern,
		upRule.TitlePattern,
		upRule.DescriptionPattern,
	)
	if err != nil {
		return nil, errors.New("can not create ubuntu planet channel: " + err.Error())
	}
	api.db.Preload("Rule").Where("ID = ?", upChannel.ID).Find(&upChannel)
	return upChannel, nil
}

func TestServerDatabase(t *testing.T) {
	deferFunc, err := StartPostgres()
	if err != nil {
		panic(err)
	}
	defer deferFunc()
	log.Println("successfully started test db in the docker")

	config, err := ParseConfig(TestConfigPath)
	if err != nil {
		panic("config parsing error: " + err.Error())
	}
	var dbApi DBApi
	dbApi.Init(&config.PostgresConfig, config.AddExamples)

	habrTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadFile("tests/data/habr.com_response")
		if err != nil {
			panic("Can not create a test server" + err.Error())
		}
		w.Write(data)
	}))
	defer habrTs.Close()

	upTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadFile("tests/data/ubuntu_planet_response")
		if err != nil {
			panic("Can not create a test server" + err.Error())
		}
		w.Write(data)
	}))
	defer habrTs.Close()

	Convey("Test database", t, func() {
		Convey("Test channel creating", func() {
			ch, err := addHabrChannel(&dbApi, habrTs.URL)

			So(err, ShouldBeNil)

			channel, err := dbApi.GetChannelById(ch.ID)
			So(err, ShouldBeNil)

			So(channel.Name, ShouldEqual, "Habr")
			So(channel.Source, ShouldEqual, habrTs.URL)
			So(channel.Rule.ItemPattern, ShouldEqual, "(?s)<article\\sclass=\"post\\spost_preview\">(.*?)</article>")
			So(channel.Rule.TitlePattern, ShouldEqual, "<a\\shref=\".*?\"\\sclass=\"post__title_link\">(.*?)</a>")
			So(channel.Rule.LinkPattern, ShouldEqual, "<a\\shref=\"(.*?)\"\\sclass=\"post__title_link\">.*?</a>")
			So(channel.Rule.DescriptionPattern, ShouldEqual, "(?s)<div\\sclass=\"post__text\\spost__text-html\\sjs-mediator-article\">(.*?)</div>\\s\\s\\s\\s\\s\\s\\s\\s\\s\\s<a class=\"btn\\sbtn_x-large\\sbtn_outline_blue\\spost__habracut-btn\"")
		})

		Convey("Test marking channel as broken", func() {
			var channels []Channel
			dbApi.db.Find(&channels)
			So(len(channels), ShouldEqual, 1)
			channel := channels[0]
			dbApi.MarkChannelAsBroken(channel.ID)
			dbApi.db.Find(&channels)
			So(len(channels), ShouldEqual, 1)
			channel = channels[0]
			So(channel.IsBroken, ShouldEqual, true)
		})

		Convey("Test deleting channel", func() {
			_, err := addUbuntuPlanetChannel(&dbApi, upTs.URL)
			So(err, ShouldBeNil)
			var channels []Channel
			dbApi.db.Where("Name = ?", "Habr").Find(&channels)
			So(len(channels), ShouldEqual, 1)
			channel := channels[0]
			dbApi.DeleteChannel(channel.ID)
			dbApi.db.Find(&channels)
			So(len(channels), ShouldEqual, 1)
			upChannel := channels[0]
			So(upChannel.Name, ShouldEqual, "Ubuntu Planet")
		})

		Convey("Test creating post", func() {
			channel, err := addHabrChannel(&dbApi, habrTs.URL)
			So(err, ShouldBeNil)

			title := "Title"
			link := "http://example.com"
			description := "Description"

			dbApi.CreatePost(title, link, description, channel.ID)
			posts := dbApi.GetChannelContent(channel.ID)
			So(len(posts), ShouldEqual, 1)
			post := posts[0]

			So(post.Link, ShouldEqual, link)
			So(post.Title, ShouldEqual, title)
			So(post.Description, ShouldEqual, description)
		})

		Convey("Test creating rule", func() {
			rule, err := dbApi.CreateRule(
				habrRule.ItemPattern,
				habrRule.LinkPattern,
				habrRule.TitlePattern,
				habrRule.DescriptionPattern,
			)
			So(err, ShouldBeNil)
			So(rule.ItemPattern, ShouldEqual, habrRule.ItemPattern)
			So(rule.TitlePattern, ShouldEqual, habrRule.TitlePattern)
			So(rule.LinkPattern, ShouldEqual, habrRule.LinkPattern)
			So(rule.DescriptionPattern, ShouldEqual, habrRule.DescriptionPattern)
		})

		Convey("Test listing channels", func() {
			dbApi.db.Unscoped().Delete(Channel{})
			channels := dbApi.ListChannels()
			So(len(channels), ShouldEqual, 0)

			expectedHabrChannel, err := addHabrChannel(&dbApi, habrTs.URL)
			So(err, ShouldBeNil)
			expectedUpChannel, err := addUbuntuPlanetChannel(&dbApi, upTs.URL)
			So(err, ShouldBeNil)

			channels = dbApi.ListChannels()
			So(len(channels), ShouldEqual, 2)

			var actualHabrChannel Channel
			dbApi.db.Where("Name = ?", "Habr").First(&actualHabrChannel)

			var actualUpChannel Channel
			dbApi.db.Where("Name = ?", "Ubuntu Planet").First(&actualUpChannel)

			So(expectedHabrChannel.Name, ShouldEqual, actualHabrChannel.Name)
			So(expectedUpChannel.Name, ShouldEqual, actualUpChannel.Name)
			So(expectedHabrChannel.Source, ShouldEqual, actualHabrChannel.Source)
			So(expectedUpChannel.Source, ShouldEqual, actualUpChannel.Source)
		})
	})

	Convey("Test content manipulations", t, func() {
		Convey("Test parsing content", func() {
			data, err := ioutil.ReadFile("tests/data/habr.com_response")
			So(err, ShouldBeNil)

			data = []byte(html.UnescapeString(string(data)))

			actualPosts, err := ParseContent(compiledHabrRule, data)
			So(err, ShouldBeNil)
			So(len(actualPosts), ShouldEqual, 20)

			expectedPosts, err := getExpectedPosts("tests/data/habr.com_posts")
			So(err, ShouldBeNil)

			for _, actualHabrPost := range actualPosts {
				actualHabrPost.CreatedAt = time.Time{}
				actualHabrPost.UpdatedAt = time.Time{}
				actualHabrPost.ID = 0
				actualHabrPost.ChannelID = 0
				So(expectedPosts, ShouldContain, actualHabrPost)
			}
		})

		Convey("Test getting content", func() {
			expectedPosts, err := getExpectedPosts("tests/data/habr.com_posts")
			So(err, ShouldBeNil)
			actualPosts, err := GetContent(habrTs.URL, compiledHabrRule)
			So(err, ShouldBeNil)

			for _, actualHabrPost := range actualPosts {
				actualHabrPost.CreatedAt = time.Time{}
				actualHabrPost.UpdatedAt = time.Time{}
				actualHabrPost.ID = 0
				actualHabrPost.ChannelID = 0
				So(expectedPosts, ShouldContain, actualHabrPost)
			}
		})

		Convey("Test fetching channel content", func() {
			channels := dbApi.ListChannels()
			for _, channel := range channels {
				err := dbApi.DeleteChannel(channel.ID)
				So(err, ShouldBeNil)
			}
			habrChannel, err := addHabrChannel(&dbApi, habrTs.URL)
			So(err, ShouldBeNil)

			upChannel, err := addUbuntuPlanetChannel(&dbApi, upTs.URL)
			So(err, ShouldBeNil)

			err = dbApi.FetchChannelContent(habrChannel)
			So(err, ShouldBeNil)

			err = dbApi.FetchChannelContent(upChannel)
			So(err, ShouldBeNil)

			expectedHabrPosts, err := getExpectedPosts("tests/data/habr.com_posts")
			So(err, ShouldBeNil)

			actualHabrPosts := dbApi.GetChannelContent(habrChannel.ID)

			So(len(actualHabrPosts), ShouldEqual, len(expectedHabrPosts))

			for _, actualHabrPost := range actualHabrPosts {
				actualHabrPost.CreatedAt = time.Time{}
				actualHabrPost.UpdatedAt = time.Time{}
				actualHabrPost.ID = 0
				actualHabrPost.ChannelID = 0
				So(expectedHabrPosts, ShouldContain, actualHabrPost)
			}

			expectedUpPosts, err := getExpectedPosts("tests/data/ubuntu_planet_posts")
			So(err, ShouldBeNil)

			actualUpPosts := dbApi.GetChannelContent(upChannel.ID)

			So(len(actualUpPosts), ShouldEqual, len(expectedUpPosts))

			for _, actualUpPost := range actualUpPosts {
				actualUpPost.CreatedAt = time.Time{}
				actualUpPost.UpdatedAt = time.Time{}
				actualUpPost.ID = 0
				actualUpPost.ChannelID = 0
				So(expectedUpPosts, ShouldContain, actualUpPost)
			}
		})

		Convey("Test removing channel content", func() {
			var habrChannel, upChannel Channel
			dbApi.db.Where("Name = ?", "Habr").First(&habrChannel)
			dbApi.db.Where("Name = ?", "Ubuntu Planet").First(&upChannel)

			dbApi.RemoveChannelContent(&habrChannel)
			posts := dbApi.GetChannelContent(habrChannel.ID)
			So(len(posts), ShouldEqual, 0)

			expectedUpPosts, err := getExpectedPosts("tests/data/ubuntu_planet_posts")
			So(err, ShouldBeNil)

			actualUpPosts := dbApi.GetChannelContent(upChannel.ID)
			So(len(actualUpPosts), ShouldEqual, len(expectedUpPosts))

			for _, actualUpPost := range actualUpPosts {
				actualUpPost.CreatedAt = time.Time{}
				actualUpPost.UpdatedAt = time.Time{}
				actualUpPost.ID = 0
				actualUpPost.ChannelID = 0
				So(expectedUpPosts, ShouldContain, actualUpPost)
			}
		})
	})
}
