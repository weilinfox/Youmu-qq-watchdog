package main

import (
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/weilinfox/youmu-qq/bot"
	"github.com/weilinfox/youmu-qq/config"
	"github.com/weilinfox/youmu-qq/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	_ "github.com/weilinfox/youmu-qq/modules/logging"
	_ "github.com/weilinfox/youmu-qq/modules/yooing"
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
		bot.SaveToken()
	} else {
		log.Panicln(err)
	}

	// re login.
	// It seems that it is useless
	var reLoginLock sync.Mutex
	bot.Instance.DisconnectedEvent.Subscribe(func(q *client.QQClient, e *client.ClientDisconnectedEvent) {
		reLoginLock.Lock()
		defer reLoginLock.Unlock()

		log.Println("Waiting for re login")
		for {
			// check internet
			resp, err := http.Get("https://tencent.com")
			if err != nil {
				time.Sleep(time.Second * 10)
				continue
			}
			_ = resp.Body.Close()
			break
		}

		log.Println("Try to login")
		err = bot.Login()
		if err != nil {
			bot.Stop()
			log.Panicln(err)
		}

		_ = os.WriteFile("session.token", bot.Instance.GenToken(), 0o644)
		log.Println("Re login success")
		bot.RefreshList()
	})

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
		log.Println(group.Name + ": " + mem.CardName + " -> " + newCard)
		mem.EditCard(newCard)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	// ↓ warn that code start here would not be run in debug mode

	// recover card name
	for i, c := range oldGroupCard {
		group := bot.Instance.FindGroup(i)
		if group == nil {
			continue
		}
		mem := group.FindMember(selfID)
		log.Println(group.Name + ": " + newCard + " -> " + c)
		mem.EditCard(c)
	}

	bot.Stop()
}
