package main

import (
	"sync"

	"github.com/linushung/hermes/cmd/server"
	"github.com/linushung/hermes/internal/app/kafkaconsumer"
	"github.com/linushung/hermes/internal/app/rabbitmqconsumer"
	"github.com/linushung/hermes/internal/pkg/configs"

	logrustash "github.com/bshuster-repo/logrus-logstash-hook"
	log "github.com/sirupsen/logrus"
	//"go.elastic.co/apm/module/apmlogrus"
)

func initLogrus() {
	logstashHost := configs.GetConfigStr("connection.logstash.host")

	if logstashHost == "" {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{
			// DisableColors: true,
			TimestampFormat: "2006-01-02 15:04:05.000",
			FullTimestamp:   true,
		})
	} else {
		log.SetFormatter(&log.JSONFormatter{})
		hook, err := logrustash.NewHookWithFields("udp", logstashHost, "hermes", log.Fields{})
		if err != nil {
			log.Fatal(err)
		}

		log.AddHook(hook)
		//log.AddHook(&apmlogrus.Hook{})
	}
}

func initService() {
	var wg sync.WaitGroup
	server.InitCircuitBreakerMgr()

	if configs.IsConfigSet("kafka") {
		wg.Add(1)
		kafkaconsumer.InitConsumerMgr().InitConsumerGroup()
	}

	if configs.IsConfigSet("rabbitmq") {
		wg.Add(1)
		rabbitmqconsumer.InitRabbitMQConnector().InitConsumerGroup()
	}

	wg.Wait()
}

func main() {
	log.Infof("***** [INIT:HERMES] ***** Start to launch Hermes 🤓 ...")
	configs.InitConfig()
	initLogrus()
	initService()
}
