package rabbitmq

import (
	"errors"
	"fmt"
	"time"

	"github.com/soedev/soelib/common/config"
	"github.com/soedev/soelib/common/soelog"
	"github.com/soedev/soelib/tools/ants"
	"github.com/spf13/cast"
	"github.com/streadway/amqp"
)

// Message 消息
type Message struct {
	Exchange    string `json:"exchange"`
	QueueName   string `json:"queueName"`
	Content     string `json:"content"`
	NotifyCount int    `json:"notifyCount"`
}

// InitRabbitMQConsumer 初始化消费者
func (c *Connection) InitRabbitMQConsumer(isClose bool, rabbitMQConfig config.Rabbit) {
	if isClose {
		c.CloseProcess <- true
	}
	c.Rabbit = rabbitMQConfig
	url := "amqp://" + c.Rabbit.Username + ":" + c.Rabbit.Password + "@" + c.Rabbit.Host + ":" + cast.ToString(c.Rabbit.Port) + "/"
	//连接rabbit
	conn, err := amqp.Dial(url)
	if err != nil {
		soelog.Logger.Error(fmt.Sprintf("rabbit连接异常:%s", err.Error()))
		soelog.Logger.Info("休息5S,开始重连rabbitMq消费者")
		time.Sleep(5 * time.Second)
		ants.SubmitTask(func() { c.InitRabbitMQConsumer(false, c.Rabbit) })
		return
	}
	defer conn.Close()
	c.Conn = conn
	err = c.CreateRabbitMQConsumer()
	if err != nil {
		soelog.Logger.Error(err.Error())
		return
	}
	c.URL = url
	c.CloseProcess = make(chan bool, 1)
	c.ConsumerReConnect()
	soelog.Logger.Info("结束消费者旧主进程")
}

func (c *Connection) CreateRabbitMQConsumer() error {
	if len(c.RabbitConsumerList) == 0 {
		return errors.New("消费者信息不能为空")
	}
	var err error
	for _, value := range c.RabbitConsumerList {
		//创建一个通道
		c.Ch, err = c.Conn.Channel()
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("MQ %s:%s", "打开Rabbit通道失败", err.Error()))
		}
		if err = c.Ch.ExchangeDeclare(
			value.ExchangeName,
			value.ExchangeType,
			true,
			false,
			false,
			false,
			nil,
		); err != nil {
			return fmt.Errorf(fmt.Sprintf("交换机初始化失败,交换机名称:%s,错误:%s", value.ExchangeName, err.Error()))
		}
		var queue amqp.Queue
		queue, err = c.Ch.QueueDeclare(
			value.QueueName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("队列初始化失败,队列名称:%s,错误: %s", value.QueueName, err.Error())
		}
		soelog.Logger.Info(fmt.Sprintf("declared Queue (%q %d messages, %d consumers), binding to Exchange (key %q)",
			queue.Name, queue.Messages, queue.Consumers, queue.Name))
		// 绑定队列
		err = c.Ch.QueueBind(value.QueueName, value.QueueName, value.ExchangeName, false, nil)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("MQ %s:%s", "绑定队列失败", err.Error()))
		}
		//绑定消费者
		messages := make(<-chan amqp.Delivery)
		messages, err = c.Ch.Consume(value.QueueName, "", false, false, false, false, nil)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("MQ %s:%s", "创建消费者失败", err.Error()))
		}
		ants.SubmitTask(func() {
			c.ConsumeHandle(messages)
		})
	}
	return nil
}

// ConsumerReConnect 消费者重连
func (c *Connection) ConsumerReConnect() {
closeTag:
	for {
		c.ConnNotifyClose = c.Conn.NotifyClose(make(chan *amqp.Error))
		c.ChNotifyClose = c.Ch.NotifyClose(make(chan *amqp.Error))
		var err *amqp.Error
		select {
		case err, _ = <-c.ConnNotifyClose:
			if err != nil {
				soelog.Logger.Error(fmt.Sprintf("rabbit消费者连接异常:%s", err.Error()))
			}
			// 判断连接是否关闭
			if !c.Conn.IsClosed() {
				if err := c.Conn.Close(); err != nil {
					soelog.Logger.Error(fmt.Sprintf("rabbit连接关闭异常:%s", err.Error()))
				}
			}
			_, isConnChannelOpen := <-c.ConnNotifyClose
			if isConnChannelOpen {
				close(c.ConnNotifyClose)
			}
			ants.SubmitTask(func() {
				c.InitRabbitMQConsumer(false, c.Rabbit)
			})
			break closeTag
		case err, _ = <-c.ChNotifyClose:
			if err != nil {
				soelog.Logger.Error(fmt.Sprintf("rabbit消费者连接异常:%s", err.Error()))
			}
			// 判断连接是否关闭
			if !c.Conn.IsClosed() {
				if err := c.Conn.Close(); err != nil {
					soelog.Logger.Error(fmt.Sprintf("rabbit连接关闭异常:%s", err.Error()))
				}
			}
			_, isConnChannelOpen := <-c.ConnNotifyClose
			if isConnChannelOpen {
				close(c.ConnNotifyClose)
			}
			ants.SubmitTask(func() {
				c.InitRabbitMQConsumer(false, c.Rabbit)
			})
			break closeTag
		case <-c.CloseProcess:
			break closeTag
		}
	}
	soelog.Logger.Info("结束消费者旧进程")
}
