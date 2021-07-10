package main

import (
	"l4d2bot/core"
	"l4d2bot/utils"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mrs4s/MiraiGo/client"
	log "github.com/sirupsen/logrus"
)

func main() {
	conf := core.LoadConfig("config.json")
	if conf.Debug {
		log.SetLevel(log.DebugLevel)
		log.Warnf("已开启Debug模式.")
	}
	log.Info("将使用 device.json 内的设备信息运行Bot.")
	client.SystemDeviceInfo.ReadJson([]byte(utils.ReadAllText("device.json")))
	log.Info("Bot将在5秒后登录并开始信息处理, 按 Ctrl+C 取消.")
	time.Sleep(time.Second * 5)
	log.Info("开始尝试登录并同步消息...")
	cli := client.NewClient(conf.Uin, conf.Password)
	rsp, err := cli.Login()
	b := core.NewBot(cli, conf)
	for {
		utils.Check(err)
		if !rsp.Success {
			switch rsp.Error {
			case client.NeedCaptcha:
				log.Fatalf("登录失败: 需要验证码. (验证码处理正在开发中)")
				continue
			case client.UnsafeDeviceError:
				log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证并重启Bot.", rsp.VerifyUrl)
				return
			case client.OtherLoginError, client.UnknownLoginError:
				log.Fatalf("登录失败: %v", rsp.ErrorMessage)
			}
		}
		break
	}
	log.Infof("登录成功 欢迎使用: %v", cli.Nickname)
	time.Sleep(time.Second)
	log.Info("开始加载好友列表...")
	utils.Check(cli.ReloadFriendList())
	log.Infof("共加载 %v 个好友.", len(cli.FriendList))
	log.Info("开始加载群列表...")
	utils.Check(cli.ReloadGroupList())
	log.Infof("共加载 %v 个群.", len(cli.GroupList))
	log.Info("资源初始化完成, 开始处理信息.")
	cli.OnDisconnected(func(bot *client.QQClient, e *client.ClientDisconnectedEvent) {
		if conf.ReLogin {
			log.Warnf("Bot已离线，将在 %v 秒后尝试重连.", conf.ReLoginDelay)
			time.Sleep(time.Second * time.Duration(conf.ReLoginDelay))
			rsp, err := cli.Login()
			if err != nil {
				log.Fatalf("重连失败: %v", err)
			}
			if !rsp.Success {
				switch rsp.Error {
				case client.NeedCaptcha:
					log.Fatalf("重连失败: 需要验证码. (验证码处理正在开发中)")
				case client.UnsafeDeviceError:
					log.Fatalf("重连失败: 设备锁")
				default:
					log.Fatalf("重连失败: %v", rsp.ErrorMessage)
				}
			}
			return
		}
		b.Close()
		log.Fatalf("Bot已离线：%v", e.Message)
	})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	b.Close()
}
