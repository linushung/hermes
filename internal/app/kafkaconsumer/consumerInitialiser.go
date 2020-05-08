package kafkaconsumer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/linushung/hermes/internal/pkg/configs"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

type consumer struct {
	Topic       string  `mapstructure:"topic"`
	GroupID     string  `mapstructure:"groupID"`
	Concurrency int     `mapstructure:"concurrency"`
	Handler     handler `mapstructure:"handler"`
}

// KafkaConfig defines Kafka configuration of hermes
type kafkaConfig struct {
	Consumers map[string]*consumer `mapstructure:"consumers"`
}

type baseConsumer struct {
	BootstrapServers []string
	MinBytes         int
	MaxBytes         int
	MaxWait          time.Duration
	ReadLagInterval  time.Duration
	SessionTimeout   time.Duration
	// HeartbeatInterval time.Duration
	GroupBalancers []kafka.GroupBalancer
}

// consumerManager defines the basic configuration of consumer for each topic
type consumerManager struct {
	kafkaConfig
	baseConsumer
}

// InitConsumerMgr prepares consumer's base configuration and channel for each topic
func InitConsumerMgr() *consumerManager {
	cons := make(map[string]*consumer)
	for _, cli := range configs.GetConfigSlice("kafka.clients") {
		con := &consumer{}
		if err := configs.GetConfigUnmarshalKey(fmt.Sprintf("kafka.consumers.%s", cli), con); err != nil {
			log.Errorf("***** [INIT:KAFKA][FAIL] ***** Failed to init consumer::%s configuration:: %v ......", cli, err)
		}

		con.Handler.Tube = make(chan *kafka.Message)
		con.Handler.HandleFunc = con.Handler.handlerDispatcher()
		cons[cli] = con
		log.Infof("***** [INIT:KAFKA] ***** Prepare consumer for client::%s ......", cli)
	}
	kconf := kafkaConfig{cons}

	baseConsumer := baseConsumer{
		BootstrapServers: strings.Split(configs.GetConfigStr("kafka.bootstrapservers"), ","),
		MinBytes:         10e3,            // 10KB
		MaxBytes:         10e6,            // 10MB
		MaxWait:          1 * time.Second, // Maximum amount of time to wait for new data to come when fetching batches of messages from kafka.
		ReadLagInterval:  -1,
		SessionTimeout:   10,
		GroupBalancers:   []kafka.GroupBalancer{kafka.RoundRobinGroupBalancer{}},
	}

	return &consumerManager{kconf, baseConsumer}
}

// ConsumerInitialiser initialise all consumers of Kafka topics
func (cmgr *consumerManager) InitConsumerGroup() {
	for cli, con := range cmgr.Consumers {
		log.Infof("***** [KAFKA:%s] ***** Init Consumer Group::%s with %d consumers for Topic::%s ......", cli, con.GroupID, con.Concurrency, con.Topic)
		for i := 1; i <= con.Concurrency; i++ {
			go con.initKafkaConsumer(cmgr.baseConsumer)
			go con.Handler.HandleFunc.Call(nil)
		}
	}
}

func (c *consumer) initKafkaConsumer(bc baseConsumer) {
	config := kafka.ReaderConfig{
		Brokers:         bc.BootstrapServers,
		GroupID:         c.GroupID,
		Topic:           c.Topic,
		MinBytes:        bc.MinBytes,
		MaxBytes:        bc.MaxBytes,
		MaxWait:         bc.MaxWait,
		ReadLagInterval: bc.ReadLagInterval,
	}

	reader := kafka.NewReader(config)
	// reader.SetOffset(kafka.LastOffset)
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Errorf("Failed to receive message from Topic::%s: %#v", msg.Topic, err.Error())
			continue
		}

		// Log message metadata received by consumer
		//log.Infof("***** [KAFKA:CONSUMER] ***** Consumer Group::%s receives message from Topic::%s ......", config.GroupID, config.Topic)
		//log.Infof("***** [KAFKA:CONSUMER] ***** Topic::%s Partition::%d Offset::%d ......", msg.Topic, msg.Partition, msg.Offset)
		c.Handler.Tube <- &msg
	}
}
