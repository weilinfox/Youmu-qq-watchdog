package yooing

import (
	"encoding/json"
	"github.com/Mrs4s/MiraiGo/message"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/CuteReimu/bilibili"
	"github.com/weilinfox/youmu-qq/bot"
	"github.com/weilinfox/youmu-qq/config"
	"github.com/weilinfox/youmu-qq/utils"
)

func init() {
	instance = &yooing{}
	bot.RegisterModule(instance)
}

var (
	instance *yooing
	logger   = utils.GetModuleLogger("internal.yooing")

	biliList     = make(map[int]int64)
	biliUserList = make(map[int]string)
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
					lastStatus[b] = info.LiveStatus == 1
					if !e {
						biliUserList[b] = getBilibiliUserName(info.Uid)
						if lastStatus[b] {
							logger.Infof("[bilibili]: %s 正在直播", biliUserList[b])
						} else {
							logger.Infof("[bilibili]: %s 在摸19诶嘿", biliUserList[b])
						}
					} else {
						if info.LiveStatus == 1 && !ls {
							msg := "【" + info.Title + "-bilibili直播】\nhttps://live.bilibili.com/" + strconv.Itoa(b) + "?broadcast_type=0&is_room_feed=1"
							bot.Instance.SendGroupMessage(q, &message.SendingMessage{
								Elements: []message.IMessageElement{&message.TextElement{Content: msg}},
							})
						}
					}
				}
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(200)+300))
			}
			time.Sleep(time.Second * time.Duration(rand.Intn(5)+2))
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

func getBilibiliUserName(uid int) string {
	type bilibiliUser struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Ttl     int    `json:"ttl"`
		Data    struct {
			Card struct {
				Name string `json:"name"`
			} `json:"card"`
		} `json:"data"`
	}

	params := url.Values{"mid": {strconv.Itoa(uid)}, "photo": {"false"}}
	resp, err := http.Get("https://api.bilibili.com/x/web-interface/card?" + params.Encode())
	if err != nil {
		logger.WithError(err).Warn("Get request failed")
		return strconv.Itoa(uid)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Warn("Read response failed")
		return strconv.Itoa(uid)
	}

	var user bilibiliUser
	err = json.Unmarshal(body, &user)
	if err != nil {
		logger.WithError(err).Warn("Unmarshal response failed")
		return strconv.Itoa(uid)
	}
	if user.Code != 0 {
		logger.WithField("Code", user.Code).Warn("Get request error")
		return strconv.Itoa(uid)
	}

	return user.Data.Card.Name
}
