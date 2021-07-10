package core

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"l4d2bot/utils"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xv-chang/rconGo/core"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

//处理- 开头命令
func (bot *QQBot) l4d2Command(msg string, groupCode int64) {

	msg = strings.TrimSpace(msg)
	if !strings.HasPrefix(msg, "-") {
		return
	}
	matchs := strings.Split(msg, " ")
	cmd := utils.GetStrArg(matchs, 0)
	switch cmd {
	case "-下载":
		go bot.DownloadCmd(groupCode, matchs)
	case "-重启":
		go bot.RestartCmd(groupCode)
	case "-服务器信息":
		go bot.GetServerInfo(groupCode)
	case "-玩家信息":
		go bot.GetPlayers(groupCode)
	case "-rcon":
		go bot.ExecRCON(groupCode, matchs)
	case "-文件地址":
		go bot.UrlCmd(groupCode, matchs)
	case "-查找地图":
		go bot.SearchMapCmd(groupCode, matchs)
	}
}

func (bot *QQBot) SearchMapCmd(groupCode int64, matchs []string) {
	mapName := utils.GetStrArg(matchs, 1)
	maps := GetMaps(mapName)
	message := "查询结果：\n"
	for _, item := range maps {
		mapKey := "M_" + item.Id
		bot.GameMapsCache.Store(mapKey, item)
		message += mapKey + " " + item.Title + "\n"
	}
	bot.SendGroupMessage(groupCode, message)
}
func (bot *QQBot) GetServerInfo(groupCode int64) {
	sq := core.NewServerQuery(bot.Config.ServerHost)
	defer sq.Close()
	info := sq.GetInfo()
	message := fmt.Sprintf("服务器名称：%v \n", info.Name)
	message += fmt.Sprintf("当前地图：%v \n", info.Map)
	message += fmt.Sprintf("当前人数：%d/%d \n", info.Players, info.MaxPlayers)
	bot.SendGroupMessage(groupCode, message)
}
func (bot *QQBot) GetPlayers(groupCode int64) {
	sq := core.NewServerQuery(bot.Config.ServerHost)
	defer sq.Close()
	players := sq.GetPlayers()
	message := fmt.Sprintf("玩家信息(%d)：\n", len(players))
	for _, player := range players {

		time := []string{}
		d := int(player.Duration)
		hour := d / 3600
		if hour > 0 {
			time = append(time, fmt.Sprintf("%dh", hour))
		}
		min := (d % 3600) / 60
		if min > 0 {
			time = append(time, fmt.Sprintf("%dm", min))
		}
		second := d % 60
		if second > 0 {
			time = append(time, fmt.Sprintf("%ds", second))
		}
		dStr := strings.Join(time, " ")
		message += fmt.Sprintf("%v\t%v\n", player.Name, dStr)
	}
	bot.SendGroupMessage(groupCode, message)
}

func (bot *QQBot) ExecRCON(groupCode int64, matchs []string) {
	client := core.NewRCONClient(bot.Config.ServerHost, bot.Config.RCONPassword)
	defer client.Close()
	err := client.SendAuth()
	if err != nil {
		bot.SendGroupMessage(groupCode, "RCON 密码错误")
		return
	}
	command := strings.Join(matchs[1:], " ")
	r2, err := client.ExecCommand(command)
	if err != nil {
		bot.SendGroupMessage(groupCode, "认证失败")
	}
	bot.SendGroupMessage(groupCode, r2)
}

func (bot *QQBot) UrlCmd(groupCode int64, matchs []string) {
	//根据文件名获取下载地址
	fileName := utils.GetStrArg(matchs, 1)
	url, err := bot.getGroupFileUrl(groupCode, fileName)
	if err != nil {
		bot.SendGroupMessage(groupCode, err.Error())
		return
	}
	bot.SendGroupMessage(groupCode, "下载地址为："+url)
}

func (bot *QQBot) donwloadFile(groupCode int64, url string, fileName string) {
	saveFileName := path.Join(bot.Config.AddonsDir, fileName)
	//判断addons目录是否存在该文件
	if utils.PathExists(saveFileName) {
		bot.SendGroupMessage(groupCode, "服务器上文件已存在"+fileName)
		return
	}
	bot.SendGroupMessage(groupCode, "正在下载文件，请耐心等待,下载地址为:"+url)
	//下载文件到addons 目录
	downloadFile(url, saveFileName)
	bot.SendGroupMessage(groupCode, "文件"+fileName+"已下载完毕")
	if strings.HasSuffix(saveFileName, ".zip") {
		bot.SendGroupMessage(groupCode, "正在解压文件"+fileName)
		unzipVPK(saveFileName, bot.Config.AddonsDir)
		bot.SendGroupMessage(groupCode, "文件"+fileName+"里的vpk文件都已解压至addons目录")
		time.Sleep(3 * time.Second)
		removeFile(saveFileName)
	}
}

func (bot *QQBot) DownloadCmd(groupCode int64, matches []string) {
	var err error
	url := utils.GetStrArg(matches, 1)
	fileName := utils.GetStrArg(matches, 2)
	if strings.HasPrefix(url, "M_") {
		mapInfo, _ := bot.GameMapsCache.Load(url)
		fileName = mapInfo.(*MapInfo).Title + ".zip"
		url = GetDownloadURL(mapInfo.(*MapInfo).Url)
	} else if !strings.HasPrefix(url, "http") {
		fileName = url
		url, err = bot.getGroupFileUrl(groupCode, url)
		if err != nil {
			bot.SendGroupMessage(groupCode, err.Error())
			return
		}
	}
	bot.donwloadFile(groupCode, url, fileName)
}

func (bot *QQBot) RestartCmd(groupCode int64) {
	err := utils.Command("systemctl restart l4d2.service")
	if err == nil {
		bot.SendGroupMessage(groupCode, "服务已重启")
	} else {
		bot.SendGroupMessage(groupCode, "服务重启错误："+err.Error())
	}
}
func downloadFile(url string, saveFileName string) {
	res, err := http.Get(url)
	if err != nil {
		log.Fatalf("下载遇到错误: %v", err)
	}
	f, err := os.Create(saveFileName)
	if err != nil {
		log.Fatalf("创建文件遇到错误: %v", err)
	}
	io.Copy(f, res.Body)
}

func unzipVPK(zipFile string, destDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()
	var decodeName string
	for _, f := range zipReader.File {
		if f.Flags == 0 {
			//如果标致位是0  则是默认的本地编码   默认为gbk
			i := bytes.NewReader([]byte(f.Name))
			decoder := transform.NewReader(i, simplifiedchinese.GB18030.NewDecoder())
			content, _ := ioutil.ReadAll(decoder)
			decodeName = string(content)
		} else {
			//如果标志为是 1 << 11也就是 2048  则是utf-8编码
			decodeName = f.Name
		}
		if strings.HasSuffix(decodeName, ".vpk") {
			newName := strings.Replace(decodeName, ".vpk", "", -1)
			newName = strings.Replace(newName, ".", "_", -1)
			newName = strings.Replace(newName, "/", "_", -1)
			newName = newName + ".vpk"
			fpath := filepath.Join(destDir, newName)
			inFile, err := f.Open()
			if err != nil {
				return err
			}
			defer inFile.Close()

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, inFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func removeFile(file string) {
	err := os.Remove(file)
	if err != nil {
		log.Fatalf("删除文件 %v 出错: %v", file, err)
	}

}
