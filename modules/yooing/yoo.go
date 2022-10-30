package yooing

import (
	"github.com/Mrs4s/MiraiGo/message"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/CuteReimu/bilibili"
	"github.com/weilinfox/youmu-qq-watchdog/bot"
	"github.com/weilinfox/youmu-qq-watchdog/config"
	"github.com/weilinfox/youmu-qq-watchdog/utils"
)

func init() {
	instance = &yooing{}
	bot.RegisterModule(instance)
}

var (
	instance *yooing
	logger   = utils.GetModuleLogger("internal.yooing")

	biliList = make(map[int]int64)
)

type yooing struct {
}

func (m *yooing) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       "internal.yooing",
		Instance: instance,
	}
}

func (m *yooing) Init() {
	// 初始化过程
	// 在此处可以进行 Module 的初始化配置
	// 如配置读取
	configMap := config.GlobalConfig.GetStringMapString("bilibili.watch")
	for b, q := range configMap {
		bid, e := strconv.ParseInt(b, 10, 32)
		if e != nil {
			logger.Warnf("ID " + b + " parse error: " + e.Error())
			continue
		}
		qid, e := strconv.ParseInt(q, 10, 64)
		if e != nil {
			logger.Warnf("ID " + b + " parse error: " + e.Error())
			continue
		}
		biliList[int(bid)] = qid
	}
}

func (m *yooing) PostInit() {
	// 第二次初始化
	// 再次过程中可以进行跨Module的动作
	// 如通用数据库等等
}

func (m *yooing) Serve(b *bot.Bot) {
	// 注册服务函数部分
}

func (m *yooing) Start(b *bot.Bot) {
	// 此函数会新开携程进行调用
	// ```go
	// 		go exampleModule.Start()
	// ```

	// 可以利用此部分进行后台操作
	// 如http服务器等等
	go func() {
		lastStatus := make(map[int]bool)
		for {
			for b, q := range biliList {
				info, err := bilibili.GetRoomInfo(b)
				if err != nil {
					logger.WithError(err).Warn("Get room info failed")
				} else {
					ls, e := lastStatus[b]
					if !e {
						lastStatus[b] = info.LiveStatus == 1
						if lastStatus[b] {
							logger.Infof("[bilibili]: %s 正在直播", info.Title)
						} else {
							logger.Infof("[bilibili]: %s 在摸19诶嘿", info.Title)
						}
					}
					if info.LiveStatus == 1 && !ls {
						msg := "[" + info.Title + "]\nhttps://live.bilibili.com/" + strconv.Itoa(b)
						bot.Instance.SendGroupMessage(q, &message.SendingMessage{
							Elements: []message.IMessageElement{&message.TextElement{Content: msg}},
						})
					} else if info.LiveStatus == 0 && ls {
						lastStatus[b] = false
					}
				}
				time.Sleep(time.Second * time.Duration(rand.Intn(5)+5))
			}
		}
	}()

}

func (m *yooing) Stop(b *bot.Bot, wg *sync.WaitGroup) {
	// 别忘了解锁
	defer wg.Done()
	// 结束部分
	// 一般调用此函数时，程序接收到 os.Interrupt 信号
	// 即将退出
	// 在此处应该释放相应的资源或者对状态进行保存
}
