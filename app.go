package main

import (
	"fmt"
	"github.com/Logiase/MiraiGo-Template/bot"
	"github.com/Logiase/MiraiGo-Template/config"
	"github.com/Logiase/MiraiGo-Template/utils"
	"os"
	"os/signal"
	"strconv"

	_ "github.com/Logiase/MiraiGo-Template/modules/logging"
)

func init() {
	utils.WriteLogToFS(utils.LogInfoLevel, utils.LogWithStack)
	config.Init()
}

func main() {
	// 使用协议
	// 不同协议可能会有部分功能无法使用
	// 在登陆前切换协议
	bot.UseProtocol(bot.IPad)

	// 快速初始化
	bot.Init()

	// 初始化 Modules
	bot.StartService()

	// 登录
	err := bot.Login()

	if err == nil {
		_ = os.WriteFile("session.token", bot.Instance.GenToken(), 0o644)
	} else {
		fmt.Println(err)
		os.Exit(1)
	}

	// 刷新好友列表，群列表
	bot.RefreshList()

	// change card name to 狐符
	oldGroupCard := make(map[int64]string)
	selfID := config.GlobalConfig.GetInt64("bot.account")
	newCard := "桜風の狐符"
	for _, i := range config.GlobalConfig.GetStringSlice("watch.group-list") {
		g, e := strconv.ParseInt(i, 10, 64)
		if e != nil {
			continue
		}
		group := bot.Instance.FindGroup(g)
		if group == nil {
			continue
		}
		mem := group.FindMember(selfID)
		oldGroupCard[g] = mem.CardName
		fmt.Println(group.Name + ": " + mem.CardName + " -> " + newCard)
		mem.EditCard(newCard)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	// recover card name
	for i, c := range oldGroupCard {
		group := bot.Instance.FindGroup(i)
		if group == nil {
			continue
		}
		mem := group.FindMember(selfID)
		fmt.Println(group.Name + ": " + newCard + " -> " + c)
		mem.EditCard(c)
	}

	bot.Stop()
}
