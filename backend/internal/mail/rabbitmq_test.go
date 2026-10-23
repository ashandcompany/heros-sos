package mail

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

// envTestRabbitMQURL désigne un courtier RabbitMQ jetable pour les tests d'intégration, par exemple
// amqp://heros:heros@127.0.0.1:5672/. Sans elle ces tests sont ignorés ; en intégration continue (CI définie),
// ils échouent plutôt que d'être ignorés en silence. Chaque test crée ses propres échanges et files, à des noms
// uniques qu'il supprime ensuite : il ne touche jamais aux files « mail.* » d'un courtier de développement.
const envTestRabbitMQURL = "HEROS_TEST_RABBITMQ_URL"

func testBrokerURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv(envTestRabbitMQURL)
	if url != "" {
		return url
	}
	if os.Getenv("CI") != "" {
		t.Fatalf("%s doit être définie en intégration continue", envTestRabbitMQURL)
	}
	t.Skipf("%s n'est pas définie : test RabbitMQ ignoré", envTestRabbitMQURL)
	return ""
}

func brokerConfig(url string) config.Mail {
	return config.Mail{
		Workers: 2, MaxAttempts: 2, RetryDelay: time.Millisecond, SendTimeout: time.Second,
		RabbitMQ: config.RabbitMQ{URL: url, ConfirmTimeout: 2 * time.Second, ReconnectDelay: 20 * time.Millisecond},
	}
}

// uniqueTopology donne à un test ses propres objets RabbitMQ et les supprime à la fin.
func uniqueTopology(t *testing.T, url string) topology {
	t.Helper()
	suffix := ids.New().String()
	suffix = suffix[len(suffix)-12:]
	topo := topology{
		exchange: "test-" + suffix + ".mail", queue: "test-" + suffix + ".send",
		deadExchange: "test-" + suffix + ".dead", deadQueue: "test-" + suffix + ".dead",
	}
	t.Cleanup(func() {
		conn, err := amqp.Dial(url)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		channel, err := conn.Channel()
		if err != nil {
			return
		}
		_, _ = channel.QueueDelete(topo.queue, false, false, false)
		_, _ = channel.QueueDelete(topo.deadQueue, false, false, false)
		_ = channel.ExchangeDelete(topo.exchange, false, false)
		_ = channel.ExchangeDelete(topo.deadExchange, false, false)
	})
	return topo
}

// withChannel ouvre une connexion à part, pour inspecter ou perturber le courtier comme le ferait un opérateur.
func withChannel(t *testing.T, url string, use func(*amqp.Channel)) {
	t.Helper()
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("connexion de contrôle : %v", err)
	}
	defer func() { _ = conn.Close() }()
	channel, err := conn.Channel()
	if err != nil {
		t.Fatalf("canal de contrôle : %v", err)
	}
	use(channel)
}

// depth retourne le nombre de messages prêts dans une file ; -1 si la file n'existe pas.
func depth(t *testing.T, url, queue string) int {
	t.Helper()
	count := -1
	withChannel(t, url, func(channel *amqp.Channel) {
		if q, err := channel.QueueDeclarePassive(queue, true, false, false, false, nil); err == nil {
			count = q.Messages
		}
	})
	return count
}

// pop retire un message d'une file (sans attente), pour le lire.
func pop(t *testing.T, url, queue string) (amqp.Delivery, bool) {
	t.Helper()
	var (
		delivery amqp.Delivery
		ok       bool
	)
	withChannel(t, url, func(channel *amqp.Channel) {
		var err error
		if delivery, ok, err = channel.Get(queue, true); err != nil {
			t.Fatalf("lecture de %s : %v", queue, err)
		}
	})
	return delivery, ok
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("délai dépassé : %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// sendEventually dépose un mail, en patientant le temps que la connexion du Publisher s'ouvre.
func sendEventually(t *testing.T, publisher *Publisher, message Message) {
	t.Helper()
	var last error
	waitFor(t, "le Publisher accepte un mail", func() bool {
		last = publisher.Send(context.Background(), message)
		return last == nil
	})
}

func closePublisher(t *testing.T, publisher *Publisher) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := publisher.Close(ctx); err != nil {
		t.Fatalf("Close : %v", err)
	}
}

// runWorker lance un Worker ; le retour de la fonction l'arrête et vérifie qu'il s'est arrêté proprement.
func runWorker(t *testing.T, worker *Worker) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	var once sync.Once
	stop = func() {
		once.Do(func() {
			cancel()
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("Worker.Run : %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Error("le worker ne s'est pas arrêté")
			}
		})
	}
	t.Cleanup(stop)
	return stop
}

// ---- Format des messages ---------------------------------------------------------------------

func TestWireMessageRoundTrip(t *testing.T) {
	message := testMessage()
	message.ReplyTo = "visiteur@example.org"
	message.ListUnsubscribe = "https://heros-sos.example/newsletter/desinscription?token=abc"

	body, err := encodeMessage(message)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeMessage(body)
	if err != nil {
		t.Fatal(err)
	}
	if got != message {
		t.Fatalf("message relu = %+v, attendu %+v", got, message)
	}
}

func TestDecodeMessageRefusesWhatIsNotAValidMail(t *testing.T) {
	for name, body := range map[string]string{
		"pas du JSON":             `not json`,
		"destinataire invalide":   `{"to":"pas-un-email","subject":"s","text":"t"}`,
		"injection d'en-tête":     `{"to":"a@b.example","subject":"s\r\nBcc: x@y.example","text":"t"}`,
		"contenu vide":            `{"to":"a@b.example","subject":"s"}`,
		"désinscription fausse":   `{"to":"a@b.example","subject":"s","text":"t","list_unsubscribe":"javascript:alert(1)"}`,
		"corps JSON incompatible": `[1,2,3]`,
	} {
		if _, err := decodeMessage([]byte(body)); !errors.Is(err, ErrInvalidMessage) {
			t.Errorf("%s : erreur = %v, attendu ErrInvalidMessage", name, err)
		}
	}
}

// ---- Publisher sans courtier ------------------------------------------------------------------

// closedPortURL est l'adresse d'un port où rien n'écoute : la connexion y est refusée aussitôt.
func closedPortURL(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return "amqp://heros:heros@" + address + "/"
}

func TestPublisherRefusesMailsWhileTheBrokerIsDown(t *testing.T) {
	publisher := NewPublisher(brokerConfig(closedPortURL(t)), discardLogger())
	defer closePublisher(t, publisher)

	started := time.Now()
	err := publisher.Send(context.Background(), testMessage())
	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("erreur = %v, attendu ErrQueueFull : l'appelant doit pouvoir patienter et réessayer", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("le refus a pris %v : il doit être immédiat, une requête HTTP ne doit pas attendre le courtier", elapsed)
	}
}

func TestPublisherValidatesBeforeTouchingTheBroker(t *testing.T) {
	publisher := NewPublisher(brokerConfig(closedPortURL(t)), discardLogger())
	defer closePublisher(t, publisher)

	bad := testMessage()
	bad.To = "pas-un-email"
	if err := publisher.Send(context.Background(), bad); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("erreur = %v, attendu ErrInvalidMessage (et non ErrQueueFull : réessayer n'y changerait rien)", err)
	}
}

func TestPublisherCloseIsPromptAndRefusesLaterMails(t *testing.T) {
	publisher := NewPublisher(brokerConfig(closedPortURL(t)), discardLogger())

	started := time.Now()
	closePublisher(t, publisher)
	closePublisher(t, publisher) // idempotent
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Close a pris %v alors que le courtier est absent", elapsed)
	}
	if err := publisher.Send(context.Background(), testMessage()); !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("erreur = %v, attendu ErrQueueClosed", err)
	}
}

// ---- Publisher et Worker avec un courtier ---------------------------------------------------

func TestMailTravelsFromPublisherToWorker(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	recorder := NewRecorder()
	runWorker(t, newWorker(recorder, cfg, topo, discardLogger()))
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)

	message := testMessage()
	message.ReplyTo = "visiteur@example.org"
	message.ListUnsubscribe = "https://heros-sos.example/newsletter/desinscription?token=abc"
	sendEventually(t, publisher, message)

	waitFor(t, "le worker a envoyé le mail", func() bool { return len(recorder.Messages()) == 1 })
	if got := recorder.Messages()[0]; got != message {
		t.Fatalf("mail reçu par le transport = %+v, attendu %+v", got, message)
	}
	waitFor(t, "la file est vide", func() bool { return depth(t, url, topo.queue) == 0 })
	if got := depth(t, url, topo.deadQueue); got != 0 {
		t.Fatalf("file morte = %d messages, attendu 0", got)
	}
}

func TestAcceptedMailsWaitInTheQueueUntilAWorkerStarts(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)

	const count = 5
	for range count {
		sendEventually(t, publisher, testMessage())
	}
	if got := depth(t, url, topo.queue); got != count {
		t.Fatalf("file = %d messages, attendu %d : un mail accepté doit être durable, worker absent ou non", got, count)
	}

	recorder := NewRecorder()
	runWorker(t, newWorker(recorder, cfg, topo, discardLogger()))
	waitFor(t, "le worker vide la file", func() bool { return len(recorder.Messages()) == count })
}

func TestWorkerRetriesFailuresThenAcknowledges(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	mailer := &flakyMailer{}
	mailer.failures.Store(1)
	runWorker(t, newWorker(mailer, cfg, topo, discardLogger()))
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)

	sendEventually(t, publisher, testMessage())

	waitFor(t, "le mail est parti à la deuxième tentative", func() bool { return mailer.sent.Load() == 1 })
	if mailer.calls.Load() != 2 {
		t.Fatalf("tentatives = %d, attendu 2", mailer.calls.Load())
	}
	waitFor(t, "le mail est acquitté", func() bool { return depth(t, url, topo.queue) == 0 })
	if got := depth(t, url, topo.deadQueue); got != 0 {
		t.Fatalf("file morte = %d messages : un mail envoyé au second essai n'y a pas sa place", got)
	}
}

func TestWorkerSendsHopelessMailsToTheDeadLetterQueue(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	mailer := &flakyMailer{}
	mailer.failures.Store(1000)
	runWorker(t, newWorker(mailer, cfg, topo, discardLogger()))
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)

	message := testMessage()
	sendEventually(t, publisher, message)

	waitFor(t, "le mail est en file morte", func() bool { return depth(t, url, topo.deadQueue) == 1 })
	if mailer.calls.Load() != int32(cfg.MaxAttempts) {
		t.Fatalf("tentatives = %d, attendu %d", mailer.calls.Load(), cfg.MaxAttempts)
	}
	if got := depth(t, url, topo.queue); got != 0 {
		t.Fatalf("file d'envoi = %d messages : le mail ne doit pas être rejoué indéfiniment", got)
	}
	dead, ok := pop(t, url, topo.deadQueue)
	if !ok {
		t.Fatal("la file morte est vide")
	}
	got, err := decodeMessage(dead.Body)
	if err != nil || got != message {
		t.Fatalf("mail en file morte = %+v (%v), attendu le mail d'origine, intact pour être rejoué", got, err)
	}
}

func TestWorkerSendsUnreadableMessagesToTheDeadLetterQueueWithoutRetrying(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	mailer := &flakyMailer{}
	runWorker(t, newWorker(mailer, cfg, topo, discardLogger()))
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)
	sendEventually(t, publisher, testMessage()) // le temps que la topologie existe
	waitFor(t, "le premier mail est parti", func() bool { return mailer.sent.Load() == 1 })

	withChannel(t, url, func(channel *amqp.Channel) {
		if err := channel.PublishWithContext(context.Background(), topo.exchange, routingSend, false, false,
			amqp.Publishing{ContentType: "text/plain", DeliveryMode: amqp.Persistent, Body: []byte("pas un mail")}); err != nil {
			t.Fatal(err)
		}
	})

	waitFor(t, "le message illisible est en file morte", func() bool { return depth(t, url, topo.deadQueue) == 1 })
	if mailer.calls.Load() != 1 {
		t.Fatalf("le transport a été appelé %d fois : un message illisible ne doit jamais l'atteindre", mailer.calls.Load())
	}
}

func TestWorkerGivesInterruptedMailsBackToTheQueue(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	cfg.SendTimeout, cfg.MaxAttempts, cfg.RetryDelay = 200*time.Millisecond, 5, time.Hour
	mailer := &blockingMailer{started: make(chan struct{}, 10), release: make(chan struct{})}
	stop := runWorker(t, newWorker(mailer, cfg, topo, discardLogger()))
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)

	sendEventually(t, publisher, testMessage())
	select {
	case <-mailer.started:
	case <-time.After(10 * time.Second):
		t.Fatal("le worker n'a pas commencé l'envoi")
	}

	stop()

	if got := depth(t, url, topo.queue); got != 1 {
		t.Fatalf("file d'envoi = %d messages, attendu 1 : un envoi interrompu par l'arrêt doit être rejoué par le prochain worker", got)
	}
	if got := depth(t, url, topo.deadQueue); got != 0 {
		t.Fatalf("file morte = %d messages : un arrêt n'est pas un échec du mail", got)
	}
}

func TestWorkerStopsPromptlyWhenIdle(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	stop := runWorker(t, newWorker(NewRecorder(), cfg, topo, discardLogger()))
	waitFor(t, "le worker est attaché à la file", func() bool { return checkWorker(cfg.RabbitMQ, topo) == nil })

	started := time.Now()
	stop()
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("l'arrêt d'un worker inactif a pris %v", elapsed)
	}
}

func TestWorkerReconnectsWhenTheBrokerIsUnreachableAtStart(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	unreachable := cfg
	unreachable.RabbitMQ.URL = closedPortURL(t)

	// Un worker lancé avant le courtier ne s'arrête pas : il réessaie, et s'arrête proprement sur demande.
	stop := runWorker(t, newWorker(NewRecorder(), unreachable, topo, discardLogger()))
	time.Sleep(100 * time.Millisecond)
	stop()
}

func TestCheckWorkerSeesWhetherAWorkerIsAttached(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)

	if err := checkWorker(cfg.RabbitMQ, topo); err == nil {
		t.Fatal("aucune file : le contrôle de santé doit échouer")
	}
	// La file existe (le Publisher la déclare) mais personne ne la consomme.
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)
	sendEventually(t, publisher, testMessage())
	err := checkWorker(cfg.RabbitMQ, topo)
	if err == nil || !strings.Contains(err.Error(), "aucun worker") {
		t.Fatalf("contrôle sans consommateur : %v, attendu « aucun worker »", err)
	}

	runWorker(t, newWorker(NewRecorder(), cfg, topo, discardLogger()))
	waitFor(t, "le contrôle de santé voit le worker", func() bool { return checkWorker(cfg.RabbitMQ, topo) == nil })
}

func TestPublishedMailsAreWrittenToDisk(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)
	sendEventually(t, publisher, testMessage())

	delivery, ok := pop(t, url, topo.queue)
	if !ok {
		t.Fatal("la file est vide")
	}
	// Sans DeliveryMode persistant, un redémarrage du courtier perdrait les mails en attente, confirmés ou non.
	if delivery.DeliveryMode != amqp.Persistent {
		t.Fatalf("DeliveryMode = %d, attendu %d (persistant)", delivery.DeliveryMode, amqp.Persistent)
	}
	if delivery.ContentType != contentTypeMail || delivery.MessageId == "" || delivery.AppId != publisherAppID {
		t.Fatalf("propriétés = %q %q %q : le type, l'identifiant et l'émetteur aident à lire la file morte",
			delivery.ContentType, delivery.MessageId, delivery.AppId)
	}
}

// brokerProxy relaie une connexion TCP vers le courtier et peut retenir ce que le courtier répond, comme un
// réseau qui ne remet plus rien : le client voit alors ses confirmations de dépôt ne plus arriver.
type brokerProxy struct {
	URL    string
	paused atomic.Bool
}

func newBrokerProxy(t *testing.T, brokerURL string) *brokerProxy {
	t.Helper()
	target := strings.TrimSuffix(strings.SplitN(brokerURL, "@", 2)[1], "/")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	proxy := &brokerProxy{URL: "amqp://heros:heros@" + listener.Addr().String() + "/"}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			client, err := listener.Accept()
			if err != nil {
				return
			}
			go proxy.relay(client, target)
		}
	}()
	return proxy
}

func (p *brokerProxy) relay(client net.Conn, target string) {
	broker, err := net.Dial("tcp", target)
	if err != nil {
		_ = client.Close()
		return
	}
	go func() {
		_, _ = io.Copy(broker, client)
		_ = broker.Close()
	}()
	buffer := make([]byte, 32*1024)
	for {
		n, err := broker.Read(buffer)
		for p.paused.Load() {
			time.Sleep(5 * time.Millisecond)
		}
		if n > 0 {
			if _, werr := client.Write(buffer[:n]); werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	_ = client.Close()
	_ = broker.Close()
}

func TestPublisherReportsSuccessOnlyOnceTheBrokerHasConfirmed(t *testing.T) {
	url := testBrokerURL(t)
	proxy := newBrokerProxy(t, url)
	topo, cfg := uniqueTopology(t, url), brokerConfig(proxy.URL)
	cfg.RabbitMQ.ConfirmTimeout = 300 * time.Millisecond
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)
	sendEventually(t, publisher, testMessage())

	proxy.paused.Store(true)
	started := time.Now()
	err := publisher.Send(context.Background(), testMessage())
	proxy.paused.Store(false)

	// Le courtier a peut-être reçu ce mail, mais tant qu'il ne l'a pas confirmé, l'appelant ne peut pas le
	// croire en sûreté : il doit réessayer (au pire un doublon, jamais une perte).
	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("erreur = %v, attendu ErrQueueFull : sans confirmation, le dépôt n'est pas acquis", err)
	}
	if elapsed := time.Since(started); elapsed < cfg.RabbitMQ.ConfirmTimeout/2 {
		t.Fatalf("le dépôt a échoué après %v : il doit attendre la confirmation du courtier", elapsed)
	}
	sendEventually(t, publisher, testMessage())
}

// ---- Pannes du courtier -----------------------------------------------------------------------

func TestPublisherRefusesMailsWhileTheBrokerBlocksPublishers(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)
	sendEventually(t, publisher, testMessage())

	// Ce que fait le courtier quand le disque est plein : « connection.blocked » (voir watch).
	publisher.blocked.Store(true)
	started := time.Now()
	err := publisher.Send(context.Background(), testMessage())
	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("erreur = %v, attendu ErrQueueFull", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("un courtier bloqué doit faire échouer le dépôt tout de suite, pas après le délai de confirmation")
	}

	publisher.blocked.Store(false)
	sendEventually(t, publisher, testMessage())
}

func TestPublisherReconnectsAfterTheConnectionDrops(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)
	sendEventually(t, publisher, testMessage())

	publisher.mu.RLock()
	conn := publisher.session.conn
	publisher.mu.RUnlock()
	_ = conn.Close() // coupure réseau, redémarrage du courtier…

	sendEventually(t, publisher, testMessage())
	if got := depth(t, url, topo.queue); got != 2 {
		t.Fatalf("file = %d messages, attendu 2 : les deux mails acceptés doivent être là", got)
	}
}

func TestPublisherRecreatesAQueueDeletedBehindItsBack(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	publisher := newPublisher(cfg, topo, discardLogger())
	defer closePublisher(t, publisher)
	sendEventually(t, publisher, testMessage())

	withChannel(t, url, func(channel *amqp.Channel) {
		if _, err := channel.QueueDelete(topo.queue, false, false, false); err != nil {
			t.Fatal(err)
		}
	})
	// Ce mail-ci n'atteint aucune file et revient : il est perdu, mais le Publisher se reconnecte et redéclare tout.
	_ = publisher.Send(context.Background(), testMessage())

	waitFor(t, "la file est recréée", func() bool { return depth(t, url, topo.queue) >= 0 })
	sendEventually(t, publisher, testMessage())
	waitFor(t, "le mail suivant est dans la file", func() bool { return depth(t, url, topo.queue) >= 1 })
}

func TestPublisherCloseWaitsForNothingWhenIdleAndStopsItsGoroutine(t *testing.T) {
	url := testBrokerURL(t)
	topo, cfg := uniqueTopology(t, url), brokerConfig(url)
	publisher := newPublisher(cfg, topo, discardLogger())
	sendEventually(t, publisher, testMessage())

	closePublisher(t, publisher)
	select {
	case <-publisher.done:
	default:
		t.Fatal("la goroutine de connexion tourne encore après Close")
	}
}
