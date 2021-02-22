package config

import (
	"encoding/json"
	"l4d2bot/utils"

	log "github.com/sirupsen/logrus"
)

type JsonConfig struct {
	Uin          int64  `json:"uin"`
	Password     string `json:"password"`
	ReLogin      bool   `json:"relogin"`
	ReLoginDelay int    `json:"relogin_delay"`
	AddonsDir    string `json:"addons_dir"`
	Debug        bool   `json:"debug"`
}

func DefaultConfig() *JsonConfig {
	return &JsonConfig{}
}
func Load(p string) *JsonConfig {
	if !utils.PathExists(p) {
		log.Warnf("尝试加载配置文件 %v 失败: 文件不存在", p)
		return nil
	}
	c := JsonConfig{}
	err := json.Unmarshal([]byte(utils.ReadAllText(p)), &c)
	if err != nil {
		log.Warnf("尝试加载配置文件 %v 时出现错误: %v", p, err)
		return nil
	}
	return &c
}
