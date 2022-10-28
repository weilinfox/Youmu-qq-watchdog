package logging

import (
	"github.com/Logiase/MiraiGo-Template/config"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"sync"
	"time"

	"github.com/Logiase/MiraiGo-Template/bot"
	"github.com/Logiase/MiraiGo-Template/utils"
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
	watchList = config.GlobalConfig.GetIntSlice("watch.group-list")
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
}

var instance *logging

var logger = utils.GetModuleLogger("internal.logging")

var (
	pokeCount int = 3
	pokeLast  int64

	watchList []int
)

func isWatchGroup(g int64) bool {
	for _, i := range watchList {
		if int64(i) == g {
			return true
		}
	}
	return false
}

func logGroupMessage(msg *message.GroupMessage) {
	if !isWatchGroup(msg.GroupCode) {
		return
	}
	name := msg.Sender.CardName
	if name == "" {
		name = msg.Sender.Nickname
	}
	logger.
		WithField("MessageID", msg.Id).
		Infof("[%s] %s: %s", msg.GroupName, name, msg.ToString())
}

func logGroupNotifyEvent(event client.INotifyEvent) {
	switch e := event.(type) {
	case *client.GroupPokeNotifyEvent:
		if !isWatchGroup(e.GroupCode) {
			return
		}
		group := bot.Instance.FindGroup(e.GroupCode)
		sender := group.FindMember(e.Sender)
		receiver := group.FindMember(e.Receiver)
		senderName := sender.CardName
		receiverName := receiver.CardName
		if senderName == "" {
			senderName = sender.Nickname
		}
		if receiverName == "" {
			receiverName = receiver.Nickname
		}
		logger.Infof("[%s] %s 戳了戳 %s", group.Name, senderName, receiverName)

		if e.Receiver == config.GlobalConfig.GetInt64("bot.account") {
			// three auto poke per hour
			if pokeCount >= 3 && time.Now().Unix()-pokeLast >= 60*60 {
				pokeCount = 0
				pokeLast = time.Now().Unix()
			}
			if pokeCount < 3 {
				if group != nil {
					member := group.FindMember(e.Sender)
					member.Poke()
					pokeCount++
				}
			}
		}
	}
}

func logGroupMuteEvent(event *client.GroupMuteEvent) {
	if !isWatchGroup(event.GroupCode) {
		return
	}
	group := bot.Instance.FindGroup(event.GroupCode)
	target := group.FindMember(event.TargetUin)
	targetName := target.CardName
	operator := group.FindMember(event.OperatorUin)
	operatorName := operator.CardName
	if targetName == "" {
		targetName = target.Nickname
	}
	if operatorName == "" {
		operatorName = operator.Nickname
	}
	logger.Infof("[%s] %s 被 %s 禁言 %dmin 惹", group.Name, targetName, operatorName, event.Time)
}

func logPrivateMessage(msg *message.PrivateMessage) {
	logger.
		WithField("from", "PrivateMessage").
		WithField("MessageID", msg.Id).
		WithField("MessageIID", msg.InternalId).
		WithField("SenderID", msg.Sender.Uin).
		WithField("Target", msg.Target).
		Info(msg.ToString())
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
		//logPrivateMessage(privateMessage)
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
