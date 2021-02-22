package core

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"l4d2bot/utils"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

//处理- 开头命令
func l4d2Command(bot *QQBot, msg string, groupCode int64) {
	r := regexp.MustCompile(`^!(\w+)(.*)`)
	matchs := r.FindStringSubmatch(msg)
	if len(matchs) < 2 {
		return
	}
	cmd := matchs[1]
	switch cmd {
	case "wget":
		if len(matchs) > 2 {
			go wgetCmd(bot, groupCode, strings.Trim(matchs[2], " "))
		}
		break
	case "restart":
		go restartCmd(bot, groupCode)
		break
	case "url":
		if len(matchs) > 2 {
			go urlCmd(bot, groupCode, strings.Trim(matchs[2], " "))
		}
	}
	return
}

func urlCmd(bot *QQBot, groupCode int64, fileName string) {
	//根据文件名获取下载地址
	url, err := bot.getGroupFileUrl(groupCode, fileName)
	if err != nil {
		bot.SendGroupMessage(groupCode, err.Error())
		return
	}
	bot.SendGroupMessage(groupCode, "下载地址为："+url)
}

func wgetCmd(bot *QQBot, groupCode int64, fileName string) {
	//根据文件名获取下载地址
	url, err := bot.getGroupFileUrl(groupCode, fileName)
	if err != nil {
		bot.SendGroupMessage(groupCode, err.Error())
		return
	}
	saveFileName := path.Join(bot.Config.AddonsDir, fileName)
	//判断addons目录是否存在该文件
	if utils.PathExists(saveFileName) {
		bot.SendGroupMessage(groupCode, "服务器上文件已存在"+fileName)
		return
	}
	bot.SendGroupMessage(groupCode, "正在下载文件，请耐心等待")
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
func restartCmd(bot *QQBot, groupCode int64) {
	err := utils.Command("sudo systemctl restart l4d2.service")
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
			newName := strings.Replace(decodeName, "/", "_", -1)
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
