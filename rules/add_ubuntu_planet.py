#!/usr/bin/python3
import requests


title = "<title>(.*?)</title>"
link = "(?s)<link>(.*?)</link>"
description = "(?s)<description>(.*?)</description>"
item = "(?s)<item>(.*?)</item>"
name = "Ubuntu Planet"
source = "http://planet.ubuntu.com/rss20.xml"

data = {
    "channel_name": name,
    "channel_source": source,
    "item_pattern": item,
    "title_pattern": title,
    "description_pattern": description,
    "link_pattern": link,
}

r = requests.post("http://0.0.0.0:8080/addchannel", data=data)

r.raise_for_status
print("Success")

