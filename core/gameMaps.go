package core

import (
	"fmt"
	"l4d2bot/utils"
	"regexp"
	"strings"
)

type MapInfo struct {
	Id    string
	Title string
	Url   string
}

func GetMaps(prefix string) []*MapInfo {
	data := map[string]string{"prefix": prefix}
	resp, _ := utils.Fetch("https://www.gamemaps.com/search/searchlist", "POST", "application/x-www-form-urlencoded", data)
	reg := regexp.MustCompile(`<a href="(.*)">[\s\S]*?class="title">(.*)</div>[\s\S]*?</a>`)
	matched := reg.FindAllStringSubmatch(resp, -1)
	maps := make([]*MapInfo, len(matched))
	for i, item := range matched {
		url := item[1]
		id := strings.Replace(url, "//www.gamemaps.com/details/", "", -1)
		maps[i] = &MapInfo{
			Id:    id,
			Url:   url,
			Title: item[2],
		}
	}
	return maps
}

func GetDownloadURL(url string) string {
	urlPath := strings.Replace(url, "details", "mirrors/download", -1)
	return fmt.Sprintf("https:%v/6", urlPath)
}
