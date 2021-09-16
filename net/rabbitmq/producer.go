package rabbitmq

import (
	"fmt"
	"github.com/soedev/soelib/common/soelog"
	"github.com/spf13/cast"
	"github.com/streadway/amqp"
	"time"
)

type Rabbit struct {
	Host     string
	Port     int
	Username string
	Password string
}
type Connection struct {
	Conn            *amqp.Connection
	Ch              *amqp.Channel
	ConnNotifyClose chan *amqp.Error
	ChNotifyClose   chan *amqp.Error
	URL             string
	Rabbit          Rabbit
	CloseProcess    chan bool
}

func dial(url string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
func (c *Connection) ReConnector() {
	closeFlag := false
closeTag:
	for {
		c.ConnNotifyClose = c.Conn.NotifyClose(make(chan *amqp.Error))
		c.ChNotifyClose = c.Ch.NotifyClose(make(chan *amqp.Error))
		select {
		case connErr, _ := <-c.ConnNotifyClose:
			soelog.Logger.Error(fmt.Sprintf("rabbit连接异常:%s", connErr.Error()))
			// 判断连接是否关闭
			if !c.Conn.IsClosed() {
				if err := c.Conn.Close(); err != nil {
					soelog.Logger.Error(fmt.Sprintf("rabbit连接关闭异常:%s", err.Error()))
				}
			}
			if conn, err := dial(c.URL); err != nil {
				soelog.Logger.Error(fmt.Sprintf("rabbit重连失败:%s", err.Error()))
				_, isConnChannelOpen := <-c.ConnNotifyClose
				if isConnChannelOpen {
					close(c.ConnNotifyClose)
				}
				//ChNotifyClose 自动关闭
				go InitRabbitMQProducer(c)
				closeFlag = true
			} else {
				ch, _ := conn.Channel()
				c.Ch = ch
				c.Conn = conn
				soelog.Logger.Info("rabbit重连成功")
			}
			// IMPORTANT: 必须清空 Notify，否则死连接不会释放
			for err := range c.ConnNotifyClose {
				println(err)
			}
		case chErr, _ := <-c.ChNotifyClose:
			soelog.Logger.Error(fmt.Sprintf("rabbit通道连接关闭:%s", chErr.Error()))
			// 重新打开一个并发服务器通道来处理消息
			if !c.Conn.IsClosed() {
				ch, err := c.Conn.Channel()
				if err != nil {
					soelog.Logger.Error(fmt.Sprintf("rabbit channel重连失败:%s", err.Error()))
					c.ChNotifyClose <- chErr
				} else {
					c.Ch = ch
				}
			}
			for err := range c.ChNotifyClose {
				println(err)
			}
		case <-c.CloseProcess:
			break closeTag
		}
		//结束进程
		if closeFlag {
			break
		}
	}
	soelog.Logger.Info("结束生产者进程")
}

//InitRabbitMQProducer 初始化生产者
func InitRabbitMQProducer(c *Connection) {
	url := "amqp://" + c.Rabbit.Username + ":" + c.Rabbit.Password + "@" + c.Rabbit.Host + ":" + cast.ToString(c.Rabbit.Port) + "/"
	conn, err := dial(url)
	if err != nil {
		soelog.Logger.Error(fmt.Sprintf("rabbitMQ连接异常:%s", err.Error()))
		soelog.Logger.Info("休息5S,开始重连rabbitMQ")
		time.Sleep(5 * time.Second)
		go InitRabbitMQProducer(c)
		return
	}
	soelog.Logger.Info("rabbitMQ生产者连接成功")
	// 打开一个并发服务器通道来处理消息
	ch, err := conn.Channel()
	if err != nil {
		soelog.Logger.Error(fmt.Sprintf("rabbitMQ打开通道异常:%s", err.Error()))
		return
	}
	c.Conn = conn
	c.URL = url
	c.Ch = ch
	c.CloseProcess = make(chan bool, 1)
	c.ReConnector()
	soelog.Logger.Info("结束rabbitMQ生产者")
	return
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
