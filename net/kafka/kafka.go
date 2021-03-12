package kafka

import (
	"github.com/Shopify/sarama"
	"github.com/soedev/soelib/common/soelog"
	"log"
	"os"
	"os/signal"
	"time"
)

//SaramaProducer 生成生产者
func SaramaProducer(config *sarama.Config, server string) (producer sarama.AsyncProducer, err error) {
	//使用配置,新建一个异步生产者
	producer, err = sarama.NewAsyncProducer([]string{server}, config)
	if err != nil {
		soelog.Logger.Info(err.Error())
		return producer, err
	}
	defer producer.AsyncClose()
	go func(p sarama.AsyncProducer) {
		select {
		case <-p.Successes():
		case fail := <-p.Errors():
			soelog.Logger.Info(fail.Err.Error())
		}
	}(producer)
	return producer, nil
}

//SaramaConsumer 生成消费者
func SaramaConsumer(server, group, topic string) (sarama.Client, sarama.OffsetManager, sarama.PartitionOffsetManager, sarama.PartitionConsumer, sarama.Consumer) {
	config := sarama.NewConfig()

	//提交offset的间隔时间，每秒提交一次给kafka
	config.Consumer.Offsets.CommitInterval = 1 * time.Second

	//设置使用的kafka版本,如果低于V0_10_0_0版本,消息中的timestrap没有作用.需要消费和生产同时配置
	config.Version = sarama.V2_0_0_0

	//consumer新建的时候会新建一个client，这个client归属于这个consumer，并且这个client不能用作其他的consumer
	consumer, err := sarama.NewConsumer([]string{server}, config)
	if err != nil {
		panic(err)
	}

	//新建一个client，为了后面offsetManager做准备
	client, err := sarama.NewClient([]string{server}, config)
	if err != nil {
		panic("client create error")
	}

	//新建offsetManager，为了能够手动控制offset
	offsetManager, err := sarama.NewOffsetManagerFromClient(group, client)
	if err != nil {
		client.Close()
		panic("offsetManager create error")
	}

	//创建一个第2分区的offsetManager，每个partition都维护了自己的offset
	partitionOffsetManager, err := offsetManager.ManagePartition(topic, 0)
	if err != nil {
		offsetManager.Close()
		client.Close()
		panic("partitionOffsetManager create error")
	}

	//sarama提供了一些额外的方法，以便我们获取broker那边的情况
	_, _ = consumer.Topics()
	_, _ = consumer.Partitions(topic)

	//第一次的offset从kafka获取(发送OffsetFetchRequest)，之后从本地获取，由MarkOffset()得来
	nextOffset, _ := partitionOffsetManager.NextOffset()

	//创建一个分区consumer，从上次提交的offset开始进行消费
	partitionConsumer, err := consumer.ConsumePartition(topic, 0, nextOffset+1)
	if err != nil {
		partitionOffsetManager.Close()
		offsetManager.Close()
		client.Close()
		if err := consumer.Close(); err != nil {
			log.Fatalln(err)
		}
		panic(err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	return client, offsetManager, partitionOffsetManager, partitionConsumer, consumer
}

//SendSarama 发送
func SendSarama(server, topic string, value []byte) {
	config := sarama.NewConfig()
	//等待服务器所有副本都保存成功后的响应
	config.Producer.RequiredAcks = sarama.WaitForAll
	//随机向partition发送消息
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	//是否等待成功和失败后的响应,只有上面的RequireAcks设置不是NoReponse这里才有用.
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	//设置使用的kafka版本,如果低于V0_10_0_0版本,消息中的timestrap没有作用.需要消费和生产同时配置
	//注意，版本设置不对的话，kafka会返回很奇怪的错误，并且无法成功发送消息
	config.Version = sarama.V2_0_0_0

	producer, err := SaramaProducer(config, server)
	if err != nil {
		soelog.Logger.Error("send sarama error:" + err.Error())
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(value),
	}
	//使用通道发送
	producer.Input() <- msg
}
