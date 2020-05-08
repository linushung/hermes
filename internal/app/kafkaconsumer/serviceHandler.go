package kafkaconsumer

import (
	"reflect"
	"strings"

	"github.com/linushung/hermes/cmd/server"
	"github.com/segmentio/kafka-go"

	log "github.com/sirupsen/logrus"
)

type handler struct {
	Handler    string   `mapstructure:"handleFuncName"`
	EndPoints  []string `mapstructure:"endPoints"`
	Tube       chan *kafka.Message
	HandleFunc reflect.Value
}

func (h handler) handlerDispatcher() reflect.Value {
	if h.Handler == "" {
		h.Handler = server.DefaultHandler
	}
	return reflect.ValueOf(h).MethodByName(h.Handler)
}

/* Below handler functions will return by reflect.Value.MethodByName() and have to be Exported methods */

func (h handler) GeneralEventHandler() {
	for {
		msg := <-h.Tube
		for _, e := range h.EndPoints {
			_, httpErr := server.GetCircuitBreakerMgr().CBHTTPPost(server.DefaultHandler, e, "application/json", msg.Value)
			if httpErr != nil {
				log.Errorf("***** [HANDLER][FAIL] ***** Receive post error from [handler::%s] [url::%s] [Error::%s]", server.DefaultHandler, e, httpErr.Error())
			}
		}
	}
}

func (h handler) NotificationServiceHandler() {
	han := strings.ToLower(h.Handler)
	for {
		msg := <-h.Tube
		for _, e := range h.EndPoints {
			_, httpErr := server.GetCircuitBreakerMgr().CBHTTPPost(han, e, "application/json", msg.Value)
			if httpErr != nil {
				log.Errorf("***** [HANDLER][FAIL] ***** Receive post error from [handler::%s] [url::%s] [Error::%s]", h.Handler, e, httpErr.Error())
			}
		}
	}
}
