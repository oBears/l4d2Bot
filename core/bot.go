package core

import (
	"errors"
	"fmt"
	"hash/crc32"
	"sync"

	"regexp"
	"strconv"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"

	log "github.com/sirupsen/logrus"
)

var matchReg = regexp.MustCompile(`\[CQ:\w+?.*?]`)
var typeReg = regexp.MustCompile(`\[CQ:(\w+)`)
var paramReg = regexp.MustCompile(`,([\w\-.]+?)=([^,\]]+)`)

type QQBot struct {
	Client        *client.QQClient
	Config        *JsonConfig
	GameMapsCache sync.Map
}
type GroupMessage struct {
	GroupId int64
	Msg     string
}

func NewBot(cli *client.QQClient, cfg *JsonConfig) *QQBot {
	bot := &QQBot{Client: cli, Config: cfg}
	bot.Client.OnGroupMessage(bot.groupMessageEvent)
	bot.Client.OnGroupInvited(bot.groupInvitedEvent)
	bot.Client.OnNewFriendRequest(bot.newFriendRequestEvent)
	return bot
}
func (bot *QQBot) newFriendRequestEvent(c *client.QQClient, e *client.NewFriendRequest) {
	e.Accept()
}
func (bot *QQBot) groupInvitedEvent(c *client.QQClient, e *client.GroupInvitedRequest) {
	e.Accept()
}
func (bot *QQBot) groupMessageEvent(c *client.QQClient, m *message.GroupMessage) {
	for _, elem := range m.Elements {
		if _, ok := elem.(*message.GroupFileElement); ok {
			return
		}
	}
	cqm := ToStringMessage(m.Elements, m.GroupCode)
	go bot.l4d2Command(cqm, m.GroupCode)
}
func ToGlobalId(code int64, msgId int32) int32 {
	return int32(crc32.ChecksumIEEE([]byte(fmt.Sprintf("%d-%d", code, msgId))))
}
func ToStringMessage(e []message.IMessageElement, code int64) (r string) {
	for _, elem := range e {
		switch o := elem.(type) {
		case *message.TextElement:
			r += o.Content
		case *message.AtElement:
			if o.Target == 0 {
				r += "[CQ:at,qq=all]"
				continue
			}
			r += fmt.Sprintf("[CQ:at,qq=%d]", o.Target)
		case *message.ReplyElement:
			r += fmt.Sprintf("[CQ:reply,id=%d]", ToGlobalId(code, o.ReplySeq))
		case *message.ForwardElement:
			r += fmt.Sprintf("[CQ:forward,id=%s]", o.ResId)
		case *message.FaceElement:
			r += fmt.Sprintf(`[CQ:face,id=%d]`, o.Index)
		case *message.ImageElement:
			r += fmt.Sprintf(`[CQ:image,file=%s,url=%s]`, o.Filename, o.Url)
		}
	}
	return
}

func (bot *QQBot) ToElement(t string, d map[string]string, group bool) (message.IMessageElement, error) {
	switch t {
	case "text":
		return message.NewText(d["text"]), nil
	case "at":
		qq := d["qq"]
		if qq == "all" {
			return message.AtAll(), nil
		}
		t, _ := strconv.ParseInt(qq, 10, 64)
		return message.NewAt(t), nil

	default:
		return nil, errors.New("unsupported cq code: " + t)
	}
}
func (bot *QQBot) ConvertStringMessage(m string, group bool) (r []message.IMessageElement) {
	i := matchReg.FindAllStringSubmatchIndex(m, -1)
	si := 0
	for _, idx := range i {
		if idx[0] > si {
			text := m[si:idx[0]]
			r = append(r, message.NewText(text))
		}
		code := m[idx[0]:idx[1]]
		si = idx[1]
		t := typeReg.FindAllStringSubmatch(code, -1)[0][1]
		ps := paramReg.FindAllStringSubmatch(code, -1)
		d := make(map[string]string)
		for _, p := range ps {
			d[p[1]] = p[2]
		}
		elem, err := bot.ToElement(t, d, group)
		if err != nil {
			continue
		}
		r = append(r, elem)
	}
	if si != len(m) {
		r = append(r, message.NewText(m[si:]))
	}
	return
}
func (bot *QQBot) SendGroupMessage(groupId int64, msg string) int32 {
	elem := bot.ConvertStringMessage(msg, false)
	m := &message.SendingMessage{Elements: elem}
	ret := bot.Client.SendGroupMessage(groupId, m)
	return ToGlobalId(ret.GroupCode, ret.Id)
}

func (bot *QQBot) getGroupFileUrl(groupId int64, filename string) (url string, err error) {
	fs, err := bot.Client.GetGroupFileSystem(groupId)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupId, err)
		return "", err
	}
	files, _, err := fs.Root()
	if err != nil {
		log.Errorf("获取群 %v 根目录文件失败: %v", groupId, err)
		return "", err
	}
	for _, file := range files {
		if file.FileName == filename {
			return bot.Client.GetGroupFileUrl(file.GroupCode, file.FileId, file.BusId), nil
		}
	}
	return "", errors.New("找不到文件" + filename)
}
func (bot *QQBot) Close() {

	bot.Client.Conn.Close()

}
