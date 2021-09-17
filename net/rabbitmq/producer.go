package rabbitmq

import (
	"encoding/json"
	"fmt"
	"github.com/soedev/soelib/common/config"
	"github.com/soedev/soelib/common/soelog"
	"github.com/soedev/soelib/tools/ants"
	"github.com/spf13/cast"
	"github.com/streadway/amqp"
	"time"
)

//Rabbit连接
type Connection struct {
	//连接
	Conn *amqp.Connection
	//通道
	Ch *amqp.Channel
	//连接异常结束
	ConnNotifyClose chan *amqp.Error
	//通道异常接收
	ChNotifyClose chan *amqp.Error
	URL           string
	Rabbit        config.Rabbit
	//用于关闭进程
	CloseProcess chan bool
	//消费者信息
	RabbitConsumerList []config.RabbitConsumerInfo
	//生产者信息
	RabbitProducerMap map[string]string
	//自定义消费者处理函数
	ConsumeHandle func(<-chan amqp.Delivery)
}

const (
	CancelOrderDelayQueue = "soesoft.cancel.order.delay.queue"
	DelayExchange         = "soesoft.delay.exchange"
)

type DLXMessage struct {
	QueueName   string `json:"queueName"`
	Content     string `json:"content"`
	NotifyCount int    `json:"notifyCount"`
}

func dial(url string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

//ProducerReConnect 生产者重连
func (c *Connection) ProducerReConnect() {
closeTag:
	for {
		c.ConnNotifyClose = c.Conn.NotifyClose(make(chan *amqp.Error))
		c.ChNotifyClose = c.Ch.NotifyClose(make(chan *amqp.Error))
		select {
		case connErr, _ := <-c.ConnNotifyClose:
			if connErr != nil {
				soelog.Logger.Error(fmt.Sprintf("rabbitMQ连接异常:%s", connErr.Error()))
			}
			// 判断连接是否关闭
			if !c.Conn.IsClosed() {
				if err := c.Conn.Close(); err != nil {
					soelog.Logger.Error(fmt.Sprintf("rabbit连接关闭异常:%s", err.Error()))
				}
			}
			//重新连接
			if conn, err := dial(c.URL); err != nil {
				soelog.Logger.Error(fmt.Sprintf("rabbit重连失败:%s", err.Error()))
				_, isConnChannelOpen := <-c.ConnNotifyClose
				if isConnChannelOpen {
					close(c.ConnNotifyClose)
				}
				//connection关闭时会自动关闭channel
				ants.Submit(func() { c.InitRabbitMQProducer(false, c.Rabbit) })
				//结束子进程
				break closeTag
			} else { //连接成功
				c.Ch, _ = conn.Channel()
				c.Conn = conn
				soelog.Logger.Info("rabbitMQ重连成功")
			}
			// IMPORTANT: 必须清空 Notify，否则死连接不会释放
			for err := range c.ConnNotifyClose {
				println(err)
			}
		case chErr, _ := <-c.ChNotifyClose:
			if chErr != nil {
				soelog.Logger.Error(fmt.Sprintf("rabbitMQ通道连接关闭:%s", chErr.Error()))
			}
			// 重新打开一个并发服务器通道来处理消息
			if !c.Conn.IsClosed() {
				ch, err := c.Conn.Channel()
				if err != nil {
					soelog.Logger.Error(fmt.Sprintf("rabbitMQ channel重连失败:%s", err.Error()))
					c.ChNotifyClose <- chErr
				} else {
					soelog.Logger.Info("rabbitMQ通道重新创建成功")
					c.Ch = ch
				}
			} else {
				_, isConnChannelOpen := <-c.ConnNotifyClose
				if isConnChannelOpen {
					close(c.ConnNotifyClose)
				}
				ants.Submit(func() { c.InitRabbitMQProducer(false, c.Rabbit) })
				break closeTag
			}
			for err := range c.ChNotifyClose {
				println(err)
			}
		case <-c.CloseProcess:
			break closeTag
		}
	}
	soelog.Logger.Info("结束旧生产者进程")
}

//InitRabbitMQProducer 初始化生产者
func (c *Connection) InitRabbitMQProducer(isClose bool, rabbitMQConfig config.Rabbit) {
	if isClose {
		c.CloseProcess <- true
	}
	c.Rabbit = rabbitMQConfig
	url := "amqp://" + c.Rabbit.Username + ":" + c.Rabbit.Password + "@" + c.Rabbit.Host + ":" + cast.ToString(c.Rabbit.Port) + "/"
	conn, err := dial(url)
	if err != nil {
		soelog.Logger.Error(fmt.Sprintf("rabbitMQ连接异常:%s", err.Error()))
		soelog.Logger.Info("休息5S,开始重连rabbitMQ生产者")
		time.Sleep(5 * time.Second)
		ants.Submit(func() { c.InitRabbitMQProducer(false, c.Rabbit) })
		return
	}
	defer conn.Close()
	soelog.Logger.Info("rabbitMQ生产者连接成功")
	// 打开一个并发服务器通道来处理消息
	ch, err := conn.Channel()
	if err != nil {
		soelog.Logger.Error(fmt.Sprintf("rabbitMQ打开通道异常:%s", err.Error()))
		return
	}
	defer ch.Close()
	c.Conn = conn
	c.URL = url
	c.Ch = ch
	c.CloseProcess = make(chan bool, 1)
	c.ProducerReConnect()
	soelog.Logger.Info("结束rabbitMQ旧生产者")
	return
}

//SendAutoCancelOrderMessage 发送消息
func (c *Connection) SendAutoCancelOrderMessage(body []byte, autoCancelTime int) error {
	m := DLXMessage{
		QueueName:   CancelOrderDelayQueue,
		Content:     string(body),
		NotifyCount: 1,
	}
	body, _ = json.Marshal(m)
	// 发布
	err := c.Ch.Publish(
		DelayExchange,         // exchange 默认模式，exchange为空
		CancelOrderDelayQueue, // routing key 默认模式路由到同名队列，即是task_queue
		false,                 // mandatory
		false,
		amqp.Publishing{
			Headers: amqp.Table{
				"x-delay": autoCancelTime * 60 * 1000,
			},
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(body),
		})
	if err != nil {
		return err
	}
	return nil
}

func (c *Connection) SendMessage(body []byte, queueName string) {
	if c.RabbitProducerMap == nil {
		soelog.Logger.Error("未初始化生产者信息")
		return
	}
	if queueName == "" {
		soelog.Logger.Error("队列名称不能为空")
		return
	}
	exchangeName := c.RabbitProducerMap[queueName]
	if exchangeName == "" {
		soelog.Logger.Error("交换机名称不能为空")
		return
	}
	m := DLXMessage{
		QueueName:   queueName,
		Content:     string(body),
		NotifyCount: 1,
	}
	body, _ = json.Marshal(m)
	// 发布
	err := c.Ch.Publish(
		exchangeName, // exchange 默认模式，exchange为空
		queueName,    // routing key 默认模式路由到同名队列，即是task_queue
		false,        // mandatory
		false,
		amqp.Publishing{
			// 持久性的发布，因为队列被声明为持久的，发布消息必须加上这个（可能不用），但消息还是可能会丢，如消息到缓存但MQ挂了来不及持久化。
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(body),
		})
	if err != nil {
		soelog.Logger.Error("rabbitMQ 发送消息失败:" + err.Error())
		return
	}
}

/*
//BindQueueForGeneralExchange 绑定交换器和队列,开启rabbit回调确认机制
func BindQueueForGeneralExchange() error {
	// 连接
	conn, err := amqp.Dial("amqp://" + setting.JsonConfig.Rabbit.Username + ":" + setting.JsonConfig.Rabbit.Password + "@" + setting.JsonConfig.Rabbit.Host + "/")
	if err != nil {
		return err
	}
	// 打开一个并发服务器通道来处理消息
	thisCh, err := conn.Channel()
	if err != nil {
		return err
	}
	//申明一个交换器
	ch = thisCh

	err = ch.ExchangeDeclare(
		GeneralOrderExchange,
		ExchangeType,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// 申明一个队列
	q, err := ch.QueueDeclare(
		GeneralOrderCreateQueue, // name
		true,                    // durable  持久性的,如果事前已经声明了该队列，不能重复声明
		false,                   // delete when unused
		false,                   // exclusive 如果是真，连接一断开，队列删除
		false,                   // no-wait
		nil,                     // arguments
	)
	if err != nil {
		return err
	}

	authQ = &q
	//绑定队列
	err = ch.QueueBind(GeneralOrderCreateQueue, GeneralOrderCreateQueue, GeneralOrderExchange, false, nil)
	if err != nil {
		return err
	}
	//注册ack回调确认监听
	if err := ch.Confirm(false); err != nil {
		return err
	} else {
		confirmation = ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	}
	return nil
}

//SendTransactionMessage 发送消息
func SendTransactionMessage(messageRecord *models.MessageRecord) error {
	var queueName, exchangeName string
	switch messageRecord.BusinessType {
	case GeneralOrderCreate:
		exchangeName = GeneralOrderExchange
		queueName = GeneralOrderCreateQueue
	}
	m := DLXMessage{
		QueueName:   queueName,
		Content:     messageRecord.Content,
		NotifyCount: 1,
	}
	body, _ := json.Marshal(m)
	if _, err := ch.QueueDeclarePassive(
		GeneralOrderCreateQueue, // name
		true,                    // duration (note: not durable)
		false,                   // auto-delete
		false,                   // exclusive
		false,                   // noWait
		nil,                     // arguments
	); err != nil {
		soelog.Logger.Info("查询队列信息失败", zap.String("错误", err.Error()))
		return err
	}
	// 发布
	err := ch.Publish(
		exchangeName, // exchange 默认模式，exchange为空
		queueName,    // routing key 默认模式路由到同名队列，即是task_queue
		false,        // mandatory
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			// 持久性的发布，因为队列被声明为持久的，发布消息必须加上这个（可能不用），但消息还是可能会丢，如消息到缓存但MQ挂了来不及持久化。
			DeliveryMode:  amqp.Persistent,
			CorrelationId: messageRecord.UID,
			Timestamp:     time.Time{},
			Body:          []byte(body),
		})

	if err != nil {
		soelog.Logger.Info("发送消息失败", zap.String("错误", err.Error()))
		return err
	}
	messageRecordM := models.MessageRecord{Db: db.GrouponDB}
	retriesNumber := 0
	//TODO 监听消息回调确认
	for {
	Publish:
		for {
			select {
			case confirmed := <-confirmation:
				//限制消息推送重试次数
				if !confirmed.Ack {
					if retriesNumber >= 5 {
						soelog.Logger.Info("超过重试次数，请手工重试")
						return nil
					}
					retriesNumber++
					messageRecordM.UpdateRetriesNumber(messageRecord.UID)
					err := ch.Publish(
						exchangeName, // exchange 默认模式，exchange为空
						queueName,    // routing key 默认模式路由到同名队列，即是task_queue
						false,        // mandatory
						false,
						amqp.Publishing{
							ContentType: "text/plain",
							// 持久性的发布，因为队列被声明为持久的，发布消息必须加上这个（可能不用），但消息还是可能会丢，如消息到缓存但MQ挂了来不及持久化。
							DeliveryMode:  amqp.Persistent,
							CorrelationId: messageRecord.UID,
							Timestamp:     time.Time{},
							Body:          []byte(body),
						})
					soelog.Logger.Info("发送消息失败", zap.String("错误", err.Error()))
					break Publish
				} else {
					fmt.Println("回调确认")
					//更新本地信息
					messageRecordM.UpdateStatus(messageRecord.UID)
					return nil
				}
			case <-time.After(2 * time.Second):
				fmt.Println("未获取到回调确认信息")
			}
		}
	}
	return nil
}

//TODO 定时任务处理未处理成功的消息
func HandleMessageRecord() {
	//查询所有未处理成功的消息
	messageRecordM := models.MessageRecord{Db: db.GrouponDB}
	messageRecordList, err := messageRecordM.GetMessageRecordList()
	if err != nil {
		return
	}
	for _, value := range messageRecordList {
		ants.Submit(func() {
			SendTransactionMessage(&value)
		})
	}
}*/
