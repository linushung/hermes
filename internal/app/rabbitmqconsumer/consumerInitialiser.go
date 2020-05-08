package rabbitmqconsumer

import (
	"fmt"
	"os"

	"github.com/linushung/hermes/internal/pkg/configs"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type rabbitMQConnector struct {
	ConsumerTag string
	Worker      int
	Channel     *amqp.Channel
	Queue       *amqp.Queue
}

// InitRabbitMQConnector initialise connection and queue
func InitRabbitMQConnector() *rabbitMQConnector {
	username := configs.GetConfigStr("rabbitmq.username")
	password := configs.GetConfigStr("rabbitmq.password")
	host := configs.GetConfigStr("rabbitmq.host")
	// The connection abstracts the socket connection, and takes care of protocol version negotiation and
	// authentication and so on for us.
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s", username, password, host))
	if err != nil {
		log.Fatalf("***** [RABBITMQ][FAIL] ***** Failed to create Connection to RabbitMQ::%s %v", host, err)
		os.Exit(1)
	}
	// defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("***** [RABBITMQ][FAIL] ***** Failed to open a Channel of RabbitMQ::%v", err)
		os.Exit(1)
	}
	// defer ch.Close()

	queueName := configs.GetConfigStr("rabbitmq.queueName")
	// Ref: https://www.rabbitmq.com/tutorials/amqp-concepts.html
	// Declaring a queue is idempotent - it will only be created if it doesn't exist already. The declaration
	// will have no effect if the queue does already exist and its attributes are the same as those in the
	// declaration. When the existing queue attributes are not the same as those in the declaration a
	// channel-level exception with code 406 (PRECONDITION_FAILED) will be raised.
	q, err := ch.QueueDeclare(
		// Queue names may be up to 255 bytes of UTF-8 characters. An AMQP 0-9-1 broker can generate a unique
		// queue name on behalf of an app. To use this feature, pass an empty string as the queue name argument.
		// The generated name will be returned to the client with queue declaration response.
		queueName, // name
		// Durable queues are persisted to disk and thus survive broker restarts. Queues that are not durable
		// are called transient.
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("***** [RABBITMQ][FAIL] ***** Failed to declare Queue of RabbitMQ::%v", err)
		os.Exit(1)
	}

	tag := configs.GetConfigStr("rabbitmq.consumerTag")
	worker := configs.GetConfigInt("rabbitmq.workers")
	return &rabbitMQConnector{tag, worker, ch, &q}
}

func (rmq *rabbitMQConnector) InitConsumerGroup() {
	qn := rmq.Queue.Name
	msg, err := rmq.Channel.Consume(
		qn,              // queue
		rmq.ConsumerTag, // consumer
		true,            // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		log.Fatalf("***** [RABBITMQ][FAIL] ***** Failed to create Consumer for Queue::%s %v", qn, err)
		os.Exit(1)
	}

	for i := 0; i < rmq.Worker; i++ {
		go func(i int) {
			log.Infof("***** [INIT:RABBITMQ] ***** Start a RabbitMQ Consumer::%s-%v ......", qn, i+1)
			for d := range msg {
				log.Printf("RabbitMQ worker[%v] - Received a message from Queue:%s %s", i+1, qn, d.Body)
			}
		}(i)
	}
}
