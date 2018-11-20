package main

import (
	"log"
	"time"
)

type ChannelsUpdater struct {
	CommonDelay        time.Duration
	InterChannelsDelay time.Duration
	DBApi              *DBApi
}

func (cu *ChannelsUpdater) Update() {
	channels := cu.DBApi.ListChannels()
	for _, channel := range channels {
		log.Printf("start update channel %v\n", channel.ID)
		err := cu.DBApi.UpdateChannelContent(channel.ID)
		if err != nil {
			log.Println("updating error: " + err.Error())
			err = cu.DBApi.MarkChannelAsBroken(channel.ID)
			log.Printf("mark channel %v as broken", channel.ID)
			if err != nil {
				log.Println("marking channel as broken error: " + err.Error())
			}
		}
		time.Sleep(cu.InterChannelsDelay)
	}
	time.Sleep(cu.CommonDelay)
}

func RunUpdater(dbApi *DBApi, commonDelay, interChannelDelay time.Duration) {
	channelsUpdater := ChannelsUpdater{DBApi: dbApi, CommonDelay: commonDelay, InterChannelsDelay: interChannelDelay}
	for {
		channelsUpdater.Update()
	}
}
