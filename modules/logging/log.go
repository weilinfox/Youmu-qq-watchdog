package logging

import (
	"encoding/json"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/weilinfox/youmu-qq/config"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/weilinfox/youmu-qq/bot"
	"github.com/weilinfox/youmu-qq/utils"
)

func init() {
	instance = &logging{}
	bot.RegisterModule(instance)
}

type logging struct {
}

func (m *logging) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       "internal.logging",
		Instance: instance,
	}
}

func (m *logging) Init() {
	// 初始化过程
	// 在此处可以进行 Module 的初始化配置
	// 如配置读取
	stringList := [][]string{config.GlobalConfig.GetStringSlice("watch.group-list"),
		config.GlobalConfig.GetStringSlice("watch.alarm-group-list"),
		config.GlobalConfig.GetStringSlice("watch.qq-list")}
	int64List := []*[]int64{
		&groupWatchList,
		&alarmGroupWatchList,
		&userWatchList,
	}
	for i, s := range stringList {
		for _, id := range s {
			i64, e := strconv.ParseInt(id, 10, 64)
			if e != nil {
				logger.Warnf("ID " + id + " parse error: " + e.Error())
				continue
			}
			*int64List[i] = append(*int64List[i], i64)
		}
	}

	// restore private message configure
	file := "./logs/" + time.Now().Format("20060102") + ".json"
	if e, _ := utils.FileExist(file); e {
		var status userStateConfig
		b, err := os.ReadFile(file)
		if err != nil {
			logger.WithError(err).Error("Read private message user status failed")
		}
		err = json.Unmarshal(b, &status)
		if err != nil {
			logger.WithError(err).Error("Recover private message user status failed")
		}

		for i, s := range status.Config {
			i64, err := strconv.ParseInt(i, 10, 64)
			if err != nil {
				logger.WithError(err).Error("State ID " + i + " parse error: " + err.Error())
			}
			userStates[i64] = s
		}
	}
}

func (m *logging) PostInit() {
	// 第二次初始化
	// 再次过程中可以进行跨Module的动作
	// 如通用数据库等等
}

func (m *logging) Serve(b *bot.Bot) {
	// 注册服务函数部分
	registerLog(b)
}

func (m *logging) Start(b *bot.Bot) {
	// 此函数会新开携程进行调用
	// ```go
	// 		go exampleModule.Start()
	// ```

	// 可以利用此部分进行后台操作
	// 如http服务器等等
}

func (m *logging) Stop(b *bot.Bot, wg *sync.WaitGroup) {
	// 别忘了解锁
	defer wg.Done()
	// 结束部分
	// 一般调用此函数时，程序接收到 os.Interrupt 信号
	// 即将退出
	// 在此处应该释放相应的资源或者对状态进行保存

	// 吐出所有缓存私聊消息
	for _, i := range userCache {
		sender := i[0].Sender.CardName
		if sender == "" {
			sender = i[0].Sender.Nickname
		}
		for _, m := range i {
			logger.
				WithField("SenderID", m.Sender.Uin).
				WithField("Target", m.Target).
				Warnf("[%s 的缓存私聊]: %s", sender, m.ToString())
		}
	}

	// store private message configure
	status := userStateConfig{
		Config: make(map[string]userState),
	}
	for i, s := range userStates {
		status.Config[strconv.FormatInt(i, 10)] = s
	}
	c, err := json.Marshal(status)
	if err != nil {
		logger.WithError(err).Error("Cover private message user status failed")
	} else {
		err = os.WriteFile("./logs/"+time.Now().Format("20060102")+".json", c, 0644)
		if err != nil {
			logger.WithError(err).Error("Write private message user status failed")
		}
	}
}

var instance *logging

var logger = utils.GetModuleLogger("internal.logging")

type userState int

const (
	appear userState = iota
	forward
	cache
	ignore
)

var (
	pokeCount = 3
	pokeLast  int64

	groupWatchList      []int64
	alarmGroupWatchList []int64 // important groups
	userWatchList       []int64

	userCache  = make(map[int64][]message.PrivateMessage)
	userStates = make(map[int64]userState)
)

type userStateConfig struct {
	Config map[string]userState `json:"config"`
}

func isWatchGroup(g int64) bool {
	for _, i := range groupWatchList {
		if i == g {
			return true
		}
	}
	return false
}

func isWatchAlarmGroup(g int64) bool {
	for _, i := range alarmGroupWatchList {
		if i == g {
			return true
		}
	}
	return false
}

func isWatchUser(g int64) bool {
	for _, i := range userWatchList {
		if i == g {
			return true
		}
	}
	return false
}

func getUserName(user interface{}) string {
	name := ""
	switch u := user.(type) {
	case *message.Sender:
		name = u.CardName
		if strings.Trim(name, " \t\n") == "" {
			name = u.Nickname
		}
	case *client.GroupMemberInfo:
		name = u.CardName
		if strings.Trim(name, " \t\n") == "" {
			name = u.Nickname
		}
	}
	return name
}

func logGroupMessage(msg *message.GroupMessage) {
	if isWatchGroup(msg.GroupCode) {
		logger.
			WithField("MessageID", msg.Id).
			Infof("[%s] %s: %s", msg.GroupName, getUserName(msg.Sender), msg.ToString())
	} else if isWatchAlarmGroup(msg.GroupCode) {
		logger.
			WithField("MessageID", msg.Id).
			Warnf("[%s] %s: %s", msg.GroupName, getUserName(msg.Sender), msg.ToString())
	}
}

func logGroupNotifyEvent(event client.INotifyEvent) {
	switch e := event.(type) {
	case *client.GroupPokeNotifyEvent:
		if !isWatchGroup(e.GroupCode) {
			return
		}
		group := bot.Instance.FindGroup(e.GroupCode)
		if group == nil {
			return
		}
		sender := group.FindMember(e.Sender)
		receiver := group.FindMember(e.Receiver)
		logger.Infof("[%s] %s 戳了戳 %s", group.Name, getUserName(sender), getUserName(receiver))

		if e.Receiver == config.GlobalConfig.GetInt64("bot.account") {
			// three auto poke per hour
			if pokeCount >= 3 && time.Now().Unix()-pokeLast >= 60*60 {
				pokeCount = 0
				pokeLast = time.Now().Unix()
			}
			if pokeCount < 2 {
				if group != nil {
					member := group.FindMember(e.Sender)
					member.Poke()
					pokeCount++
				}
			} else if pokeCount == 2 {
				bot.Instance.SendGroupMessage(e.GroupCode, &message.SendingMessage{
					Elements: []message.IMessageElement{&message.GroupImageElement{
						ImageId:   "988EA892525938442F24EE16BC726E1F.jpg",
						FileId:    -1376696330,
						Md5:       []byte{0x98, 0x8E, 0xA8, 0x92, 0x52, 0x59, 0x38, 0x44, 0x2F, 0x24, 0xEE, 0x16, 0xBC, 0x72, 0x6E, 0x1F},
						Size:      2842,
						ImageType: 0,
						Width:     80,
						Height:    80,
						Url:       "https://gchat.qpic.cn/gchatpic_new/1/0-0-988EA892525938442F24EE16BC726E1F/0?term=2",
						Flash:     false,
					}},
				})
				pokeCount++
			}
		}
	}
}

func logGroupMuteEvent(event *client.GroupMuteEvent) {
	if !isWatchGroup(event.GroupCode) {
		return
	}
	group := bot.Instance.FindGroup(event.GroupCode)
	if group == nil {
		return
	}
	target := group.FindMember(event.TargetUin)
	operator := group.FindMember(event.OperatorUin)
	logger.Infof("[%s] %s 被 %s 禁言 %ds 惹", group.Name, getUserName(target), getUserName(operator), event.Time)
}

func logPrivateMessage(msg *message.PrivateMessage) {
	if !isWatchUser(msg.Sender.Uin) {
		// 非列表 自动battle
		if state, e := userStates[msg.Sender.Uin]; e {
			switch state {
			case appear:
				bot.Instance.MarkPrivateMessageReaded(msg.Sender.Uin, int64(msg.Time))

				imp, e := strconv.ParseInt(msg.ToString(), 10, 64)
				if e == nil {
					// 重要级别
					if imp > 5 {
						// 切换实时转发
						userStates[msg.Sender.Uin] = forward

						for _, i := range userCache[msg.Sender.Uin] {
							logger.
								WithField("SenderID", i.Sender.Uin).
								WithField("Target", i.Target).
								Warnf("[%s 的缓存私聊]: %s", getUserName(msg.Sender), i.ToString())
						}
						delete(userCache, msg.Sender.Uin)
					} else {
						// 继续缓存
						userStates[msg.Sender.Uin] = cache
						userCache[msg.Sender.Uin] = append(userCache[msg.Sender.Uin], *msg)
					}
					bot.Instance.SendPrivateMessage(msg.Sender.Uin, &message.SendingMessage{
						Elements: []message.IMessageElement{&message.TextElement{
							Content: "[狐符]\n" +
								"判断完成",
						}},
					})
				} else {
					// 忽略
					bot.Instance.SendPrivateMessage(msg.Sender.Uin, &message.SendingMessage{
						Elements: []message.IMessageElement{&message.TextElement{
							Content: "[狐符]\n" +
								"判断失败",
						}},
					})

					userStates[msg.Sender.Uin] = ignore
					delete(userCache, msg.Sender.Uin)
				}
				return
			case cache:
				bot.Instance.MarkPrivateMessageReaded(msg.Sender.Uin, int64(msg.Time))
				userCache[msg.Sender.Uin] = append(userCache[msg.Sender.Uin], *msg)
				return
			case ignore:
				bot.Instance.MarkPrivateMessageReaded(msg.Sender.Uin, int64(msg.Time))
				return
			case forward:
			}
		} else {
			bot.Instance.MarkPrivateMessageReaded(msg.Sender.Uin, int64(msg.Time))

			bot.Instance.SendPrivateMessage(msg.Sender.Uin, &message.SendingMessage{
				Elements: []message.IMessageElement{&message.TextElement{
					Content: "[狐符]\n" +
						"少女缓存中……",
				}},
			})
			bot.Instance.SendPrivateMessage(msg.Sender.Uin, &message.SendingMessage{
				Elements: []message.IMessageElement{&message.TextElement{
					Content: "[狐符]\n" +
						"在此条消息后的第一条消息，回复 ∈ [0, 10] 的一个整数，作为消息重要性的参照（越大越紧急）\n" +
						"狐符将以此为依据判断是否上报。",
				}},
			})

			userStates[msg.Sender.Uin] = appear
			userCache[msg.Sender.Uin] = []message.PrivateMessage{*msg}

			return
		}
	}

	logger.
		WithField("SenderID", msg.Sender.Uin).
		WithField("Target", msg.Target).
		Warnf("[%s 的私聊]: %s", getUserName(msg.Sender), msg.ToString())
}

func logFriendNotifyEvent(event client.INotifyEvent) {
}

func logFriendMessageRecallEvent(event *client.FriendMessageRecalledEvent) {
	logger.
		WithField("from", "FriendsMessageRecall").
		WithField("MessageID", event.MessageId).
		WithField("SenderID", event.FriendUin).
		Info("friend message recall")
}

func logGroupMessageRecallEvent(event *client.GroupMessageRecalledEvent) {
	logger.
		WithField("from", "GroupMessageRecall").
		WithField("MessageID", event.MessageId).
		WithField("GroupCode", event.GroupCode).
		WithField("SenderID", event.AuthorUin).
		WithField("OperatorID", event.OperatorUin).
		Info("group message recall")
}

func logDisconnect(event *client.ClientDisconnectedEvent) {
	logger.
		WithField("from", "Disconnected").
		WithField("reason", event.Message).
		Warn("bot disconnected")
}

func registerLog(b *bot.Bot) {
	/*b.GroupMessageRecalledEvent.Subscribe(func(qqClient *client.QQClient, event *client.GroupMessageRecalledEvent) {
		logGroupMessageRecallEvent(event)
	})*/

	b.GroupMessageEvent.Subscribe(func(qqClient *client.QQClient, groupMessage *message.GroupMessage) {
		logGroupMessage(groupMessage)
	})

	b.SelfGroupMessageEvent.Subscribe(func(qqClient *client.QQClient, groupMessage *message.GroupMessage) {
		logGroupMessage(groupMessage)
	})

	b.GroupMuteEvent.Subscribe(func(qqClient *client.QQClient, event *client.GroupMuteEvent) {
		logGroupMuteEvent(event)
	})

	b.PrivateMessageEvent.Subscribe(func(qqClient *client.QQClient, privateMessage *message.PrivateMessage) {
		logPrivateMessage(privateMessage)
	})

	b.FriendMessageRecalledEvent.Subscribe(func(qqClient *client.QQClient, event *client.FriendMessageRecalledEvent) {
		//logFriendMessageRecallEvent(event)
	})

	b.DisconnectedEvent.Subscribe(func(qqClient *client.QQClient, event *client.ClientDisconnectedEvent) {
		logDisconnect(event)
	})

	b.GroupNotifyEvent.Subscribe(func(qqClient *client.QQClient, event client.INotifyEvent) {
		logGroupNotifyEvent(event)
	})

	/*b.FriendNotifyEvent.Subscribe(func(qqClient *client.QQClient, event client.INotifyEvent) {
		logFriendNotifyEvent(event)
	})*/
}
