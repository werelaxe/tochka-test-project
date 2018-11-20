#!/usr/bin/python3
import requests


title = "<a\\shref=\".*?\"\\sclass=\"post__title_link\">(.*?)</a>"
link = "<a\\shref=\"(.*?)\"\\sclass=\"post__title_link\">.*?</a>"
description = "(?s)<div\\sclass=\"post__text\\spost__text-html\\sjs-mediator-article\">(.*?)</div>\\s\\s\\s\\s\\s\\s\\s\\s\\s\\s<a class=\"btn\\sbtn_x-large\\sbtn_outline_blue\\spost__habracut-btn\""
item = "(?s)<article\\sclass=\"post\\spost_preview\">(.*?)</article>"
name = "Habr"
source = "http://habr.com"

data = {
    "channel_name": name,
    "channel_source": source,
    "item_pattern": item,
    "title_pattern": title,
    "description_pattern": description,
    "link_pattern": link,
}

r = requests.post("http://0.0.0.0:8080/addchannel", data=data)

r.raise_for_status()
print("Success")

