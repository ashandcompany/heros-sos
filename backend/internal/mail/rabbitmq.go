package mail

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

// Les mails passent par RabbitMQ en trois temps (voir stack/rabbitmq.md) :
//
//	API  --Publisher-->  exchange « mail »  -->  file « mail.send »  --Worker-->  Mailer (SMTP)
//	                                                    |
//	                                      refus définitif (après MaxAttempts)
//	                                                    v
//	                                      exchange « mail.dead »  -->  file « mail.dead »
//
// La file morte garde les mails que le worker n'a pas pu envoyer : un humain les examine, puis les rejoue
// ou les supprime (interface de gestion, http://localhost:15672 en développement).

// topology nomme les objets RabbitMQ. Le Publisher et le Worker la déclarent tous deux à chaque connexion :
// la déclaration est idempotente, et l'ordre de démarrage des deux processus n'a pas d'importance.
type topology struct {
	exchange     string // où l'API dépose les mails
	queue        string // les mails à envoyer
	deadExchange string // où RabbitMQ redirige les mails refusés pour de bon
	deadQueue    string // la file morte
}

const (
	routingSend = "send"
	routingDead = "dead"

	contentTypeMail = "application/json"
	// deliveryPersistent écrit le message sur disque : il survit à un redémarrage de RabbitMQ.
	deliveryPersistent = amqp.Persistent
)

func defaultTopology() topology {
	return topology{exchange: "mail", queue: "mail.send", deadExchange: "mail.dead", deadQueue: "mail.dead"}
}

// declare crée les échanges, les files et leurs liaisons. Les files sont durables, comme leurs échanges.
func (t topology) declare(channel *amqp.Channel) error {
	if err := channel.ExchangeDeclare(t.deadExchange, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return fmt.Errorf("mail : échange %s : %w", t.deadExchange, err)
	}
	if _, err := channel.QueueDeclare(t.deadQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("mail : file %s : %w", t.deadQueue, err)
	}
	if err := channel.QueueBind(t.deadQueue, routingDead, t.deadExchange, false, nil); err != nil {
		return fmt.Errorf("mail : liaison de %s : %w", t.deadQueue, err)
	}

	if err := channel.ExchangeDeclare(t.exchange, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return fmt.Errorf("mail : échange %s : %w", t.exchange, err)
	}
	args := amqp.Table{"x-dead-letter-exchange": t.deadExchange, "x-dead-letter-routing-key": routingDead}
	if _, err := channel.QueueDeclare(t.queue, true, false, false, false, args); err != nil {
		return fmt.Errorf("mail : file %s : %w", t.queue, err)
	}
	if err := channel.QueueBind(t.queue, routingSend, t.exchange, false, nil); err != nil {
		return fmt.Errorf("mail : liaison de %s : %w", t.queue, err)
	}
	return nil
}

// dialBroker ouvre une connexion AMQP. name apparaît dans l'interface de gestion de RabbitMQ, pour savoir
// quel processus tient quelle connexion. Les erreurs ne contiennent pas l'URL (elle porte un mot de passe).
func dialBroker(cfg config.RabbitMQ, name string) (*amqp.Connection, error) {
	properties := amqp.NewConnectionProperties()
	properties.SetClientConnectionName("heros-sos " + name)
	return amqp.DialConfig(cfg.URL, amqp.Config{
		Dial:       amqp.DefaultDial(cfg.ConfirmTimeout),
		Properties: properties,
	})
}

// wireMessage est la forme d'un mail dans la file. Message n'est pas sérialisé directement : les noms JSON sont
// explicites pour qui lit la file morte, et ils ne changent pas avec la structure Go. Les champs sont ceux de
// Message, dans le même ordre : les conversions ci-dessous ne compilent plus si l'un des deux évolue, ce qui force
// à décider du format d'échange (un worker plus ancien relit les mails d'une API plus récente).
type wireMessage struct {
	To              string `json:"to"`
	ReplyTo         string `json:"reply_to,omitempty"`
	Subject         string `json:"subject"`
	Text            string `json:"text,omitempty"`
	HTML            string `json:"html,omitempty"`
	ListUnsubscribe string `json:"list_unsubscribe,omitempty"`
}

func encodeMessage(message Message) ([]byte, error) {
	return json.Marshal(wireMessage(message))
}

// decodeMessage relit un mail de la file. Un message illisible, ou qui n'est pas un mail valide, est
// retourné en ErrInvalidMessage : le réessayer ne le rendrait pas meilleur.
func decodeMessage(body []byte) (Message, error) {
	var wire wireMessage
	if err := json.Unmarshal(body, &wire); err != nil {
		return Message{}, fmt.Errorf("%w : %w", ErrInvalidMessage, err)
	}
	message := Message(wire)
	if _, err := message.validate(); err != nil {
		return Message{}, err
	}
	return message, nil
}
