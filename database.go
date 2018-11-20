package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"html"
	"log"
	"time"
)

const PostsBlockSize = 5

type PostgresConfig struct {
	DBName   string
	Host     string
	Port     uint
	User     string
	Password string
}

type Post struct {
	gorm.Model
	Link        string
	Title       string
	Description string
	Channel     Channel
	ChannelID   uint
}

type Rule struct {
	gorm.Model
	TitlePattern       string
	ItemPattern        string
	LinkPattern        string
	DescriptionPattern string
}

type Channel struct {
	gorm.Model
	Name     string
	Source   string
	Rule     Rule
	RuleID   uint
	IsBroken bool
}

type DBApi struct {
	db *gorm.DB
}

func (api *DBApi) CreatePost(title, link, description string, channelId uint) {
	api.db.Create(&Post{Link: link, Title: title, Description: description, ChannelID: channelId})
}

func (api *DBApi) CreateRule(itemPattern, linkPattern, titlePattern, descriptionPattern string) (*Rule, error) {
	rule := Rule{ItemPattern: itemPattern,
		LinkPattern:        linkPattern,
		TitlePattern:       titlePattern,
		DescriptionPattern: descriptionPattern,
	}
	_, err := CompileRule(&rule)
	if err != nil {
		return nil, errors.New("db error: " + err.Error())
	}
	return api.db.Create(&rule).Value.(*Rule), nil
}

func (api *DBApi) CreateChannel(name, source, itemPattern, linkPattern, titlePattern, descriptionPattern string) (*Channel, error) {
	rule, err := api.CreateRule(itemPattern, linkPattern, titlePattern, descriptionPattern)
	if err != nil {
		return nil, errors.New("db error: " + err.Error())
	}
	channel := Channel{RuleID: rule.ID, Name: name, Source: source, IsBroken: false}
	return api.db.Create(&channel).Value.(*Channel), nil
}

func (api *DBApi) MarkChannelAsBroken(channelId uint) error {
	var channels []Channel
	api.db.Where("ID = ?", channelId).Find(&channels)
	if len(channels) != 1 {
		return errors.New(fmt.Sprintf("db error, empty or multiple channels by ID=%v", channelId))
	}
	channel := channels[0]
	api.db.Model(channel).Where("ID = ?", channel.ID).Update("IsBroken", true)
	return nil
}

func (api *DBApi) DeleteChannel(channelId uint) error {
	var channels []Channel
	api.db.Where("ID = ?", channelId).Find(&channels)
	if len(channels) != 1 {
		return errors.New(fmt.Sprintf("db error, empty or multiple channels by ID=%v", channelId))
	}
	channel := channels[0]
	api.db.Unscoped().Delete(&channel)
	return nil
}

func (api *DBApi) ListChannels() []Channel {
	var channels []Channel
	api.db.Preload("Rule").Find(&channels)
	return channels
}

func (api *DBApi) RemoveChannelContent(channel *Channel) {
	api.db.Unscoped().Where("channel_id = ?", channel.ID).Delete(Post{})
}

func (api *DBApi) FetchChannelContent(channel *Channel) error {
	rule, err := CompileRule(&channel.Rule)
	if err != nil {
		return errors.New(fmt.Sprintf("db error, channel ID=%v, error=%s", channel.ID, err.Error()))
	}
	posts, err := GetContent(channel.Source, rule)
	if err != nil {
		return errors.New(fmt.Sprintf("db error, channel ID=%v, error=%s", channel.ID, err.Error()))
	}
	if err != nil {
		return errors.New(fmt.Sprintf("db error, channel ID=%v, error=%s", channel.ID, err.Error()))
	}
	for _, post := range posts {
		dbApi.CreatePost(post.Title, post.Link, html.UnescapeString(post.Description), channel.ID)
	}
	return nil
}

func (api *DBApi) GetChannelById(channelId uint) (*Channel, error) {
	var channels []Channel
	api.db.Preload("Rule").Where("ID = ?", channelId).Find(&channels)
	if len(channels) != 1 {
		log.Println(channels)
		return nil, errors.New(fmt.Sprintf("db error, empty or multiple channels by ID=%v", channelId))
	}
	return &channels[0], nil
}

func (api *DBApi) UpdateChannelContent(channelId uint) error {
	channel, err := api.GetChannelById(channelId)
	if err != nil {
		return errors.New("getting channel error: " + err.Error())
	}
	api.RemoveChannelContent(channel)
	err = api.FetchChannelContent(channel)
	if err != nil {
		return errors.New("fetching channel content error: " + err.Error())
	}
	return nil
}

func (api *DBApi) GetChannelContentWithLimit(channelId, offset, limit uint, filter string) []Post {
	var channel Channel
	api.db.Where("ID = ?", channelId).First(&channel)
	var posts []Post
	fmtFilter := fmt.Sprintf("%%%v%%", filter)
	api.db.Model(&channel).Offset(offset).Limit(limit).Where("title ILIKE ?", fmtFilter).Related(&posts, "Post")
	return posts
}

func (api *DBApi) GetChannelContent(channelId uint) []Post {
	var channel Channel
	api.db.Where("ID = ?", channelId).First(&channel)
	var posts []Post
	api.db.Model(&channel).Related(&posts, "Post")
	return posts
}

func IsDbExists(db *sql.DB, config *PostgresConfig) (bool, error) {
	result, err := db.Query("SELECT datname FROM pg_catalog.pg_database WHERE datname = '" + config.DBName + "';")
	if err != nil {
		return false, err
	}
	return result.Next(), nil
}

func CreateDbIfNotExists(config *PostgresConfig) error {
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s sslmode=disable", config.Host, config.Port, config.User))
	if err != nil {
		return err
	}
	isDbExists, err := IsDbExists(db, config)
	if err != nil {
		return err
	}
	if isDbExists {
		return nil
	}
	_, err = db.Exec("CREATE DATABASE " + config.DBName + ";")
	if err != nil {
		return err
	}
	return nil
}

func CreateDbIfNotExistsWithRetry(config *PostgresConfig, maxWait time.Duration) error {
	done := time.Now().Add(maxWait)
	for time.Now().Before(done) {
		err := CreateDbIfNotExists(config)
		if err != nil {
			log.Println("cannot create database: " + err.Error())
		} else {
			log.Println("database has been successfully created")
			return nil
		}
		time.Sleep(1000 * time.Millisecond)
	}
	return fmt.Errorf("cannot create database")
}

func AddExampleChannels(api *DBApi) error {
	if len(api.ListChannels()) != 0 {
		return nil
	}
	_, err := api.CreateChannel(
		"Habr",
		"https://habr.com",
		"(?s)<article\\sclass=\"post\\spost_preview\">(.*?)</article>",
		"<a\\shref=\"(.*?)\"\\sclass=\"post__title_link\">.*?</a>",
		"<a\\shref=\".*?\"\\sclass=\"post__title_link\">(.*?)</a>",
		"(?s)<div\\sclass=\"post__text\\spost__text-html\\sjs-mediator-article\">(.*?)</div>\\s\\s\\s\\s\\s\\s\\s\\s\\s\\s<a class=\"btn\\sbtn_x-large\\sbtn_outline_blue\\spost__habracut-btn\"",
	)
	if err != nil {
		return errors.New("Can not create Habr channel: " + err.Error())
	}
	_, err = api.CreateChannel(
		"Ubuntu Planet",
		"http://planet.ubuntu.com/rss20.xml",
		"(?s)<item>(.*?)</item>",
		"(?s)<link>(.*?)</link>",
		"<title>(.*?)</title>",
		"(?s)<description>(.*?)</description>",
	)
	if err != nil {
		return errors.New("Can not create Ubuntu Planet channel: " + err.Error())
	}
	return nil
}

func (api *DBApi) Init(config *PostgresConfig, addExamples bool) {
	err := CreateDbIfNotExistsWithRetry(config, time.Second * 10)
	if err != nil {
		panic(fmt.Sprintf("failed to create a database: %s", err))
	}
	api.db, err = gorm.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable", config.Host, config.Port, config.User, config.DBName))
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s", err))
	}
	api.db.AutoMigrate(&Post{})
	api.db.AutoMigrate(&Rule{})
	api.db.AutoMigrate(&Channel{})
	if !addExamples {
		return
	}
	err = AddExampleChannels(api)
	if err != nil {
		panic("adding examples error: " + err.Error())
	}
}
