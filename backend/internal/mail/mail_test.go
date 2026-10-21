package mail

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

var (
	testNow    = time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	testSender = Sender{Address: "no-reply@heros-sos.example", Name: "Heros-SOS"}
)

func testMessage() Message {
	return Message{
		To:      "steve@avengers.example",
		Subject: "Activez votre compte Heros-SOS — éléphant",
		Text:    "Bonjour Éloïse,\nle lien : https://heros-sos.example/activation?token=abc\n",
		HTML:    "<p>Bonjour Éloïse</p>",
	}
}

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// ---- Construction du message ----------------------------------------------------------------

func TestBuildProducesDecodableMultipartMessage(t *testing.T) {
	raw, err := build(testSender, testMessage(), testNow)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("message illisible : %v", err)
	}

	if from, _ := parsed.Header.AddressList("From"); len(from) != 1 || from[0].Address != "no-reply@heros-sos.example" || from[0].Name != "Heros-SOS" {
		t.Errorf("From = %v", from)
	}
	if to, _ := parsed.Header.AddressList("To"); len(to) != 1 || to[0].Address != "steve@avengers.example" {
		t.Errorf("To = %v", to)
	}
	decoded, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	if err != nil || decoded != testMessage().Subject {
		t.Errorf("Subject = %q (%v)", decoded, err)
	}
	if date, err := parsed.Header.Date(); err != nil || !date.Equal(testNow) {
		t.Errorf("Date = %v (%v)", date, err)
	}
	if id := parsed.Header.Get("Message-ID"); !strings.HasSuffix(id, "@heros-sos.example>") {
		t.Errorf("Message-ID = %q", id)
	}

	mediaType, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/alternative" {
		t.Fatalf("Content-Type = %q (%v)", mediaType, err)
	}
	reader := multipart.NewReader(parsed.Body, params["boundary"])
	var contents []string
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(quotedprintable.NewReader(part))
		contents = append(contents, part.Header.Get("Content-Type")+"|"+string(body))
	}
	if len(contents) != 2 ||
		!strings.HasPrefix(contents[0], "text/plain; charset=utf-8|Bonjour Éloïse") ||
		!strings.HasPrefix(contents[1], "text/html; charset=utf-8|<p>Bonjour Éloïse") {
		t.Errorf("parties = %q", contents)
	}
}

func TestMessageValidationBlocksHeaderInjection(t *testing.T) {
	tests := map[string]Message{
		"saut de ligne dans le sujet":        {To: "a@b.co", Subject: "ok\r\nBcc: espion@evil.example", Text: "x"},
		"saut de ligne dans le destinataire": {To: "a@b.co\r\nBcc: espion@evil.example", Subject: "ok", Text: "x"},
		"nom d'affichage":                    {To: "Pirate <a@b.co>", Subject: "ok", Text: "x"},
		"destinataire invalide":              {To: "pas-un-email", Subject: "ok", Text: "x"},
		"destinataire vide":                  {To: "", Subject: "ok", Text: "x"},
		"sujet vide":                         {To: "a@b.co", Subject: "  ", Text: "x"},
		"contenu vide":                       {To: "a@b.co", Subject: "ok"},
		"saut de ligne dans la réponse":      {To: "a@b.co", ReplyTo: "v@b.co\r\nBcc: espion@evil.example", Subject: "ok", Text: "x"},
		"nom d'affichage dans la réponse":    {To: "a@b.co", ReplyTo: "Pirate <v@b.co>", Subject: "ok", Text: "x"},
		"réponse invalide":                   {To: "a@b.co", ReplyTo: "pas-un-email", Subject: "ok", Text: "x"},
	}
	for name, message := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := build(testSender, message, testNow); !errors.Is(err, ErrInvalidMessage) {
				t.Fatalf("erreur = %v, attendu ErrInvalidMessage", err)
			}
		})
	}
}

func TestBuildAddsReplyToOnlyWhenGiven(t *testing.T) {
	withReply := testMessage()
	withReply.ReplyTo = "visiteur@example.org"
	raw, err := build(testSender, withReply, testNow)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if replyTo, _ := parsed.Header.AddressList("Reply-To"); len(replyTo) != 1 || replyTo[0].Address != "visiteur@example.org" {
		t.Errorf("Reply-To = %v", replyTo)
	}

	raw, err = build(testSender, testMessage(), testNow)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "Reply-To") {
		t.Error("sans adresse de réponse, aucun en-tête Reply-To ne doit être écrit")
	}
}

// ---- Serveur SMTP de test ----------------------------------------------------------------------

type receivedMail struct {
	from, to string
	data     string
}

type fakeSMTP struct {
	listener  net.Listener
	tlsConfig *tls.Config
	startTLS  bool
	implicit  bool

	mu       sync.Mutex
	received []receivedMail
}

func newFakeSMTP(t *testing.T, mode config.TLSMode) (*fakeSMTP, config.SMTP, *tls.Config) {
	t.Helper()
	server := &fakeSMTP{startTLS: mode == config.TLSModeStartTLS, implicit: mode == config.TLSModeImplicit}

	var clientTLS *tls.Config
	if mode != config.TLSModeNone {
		cert, pool := selfSignedCertificate(t)
		server.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
		clientTLS = &tls.Config{RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS12}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server.listener = listener
	t.Cleanup(func() { _ = listener.Close() })
	go server.serve()

	address := listener.Addr().(*net.TCPAddr)
	return server, config.SMTP{Host: "localhost", Port: address.Port, TLSMode: mode, Timeout: 5 * time.Second}, clientTLS
}

func (s *fakeSMTP) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeSMTP) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	if s.implicit {
		conn = tls.Server(conn, s.tlsConfig)
	}
	reader := bufio.NewReader(conn)
	reply := func(text string) { _, _ = io.WriteString(conn, text+"\r\n") }
	reply("220 fake ESMTP")

	var current receivedMail
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"):
			if s.startTLS {
				reply("250-fake\r\n250-STARTTLS\r\n250 8BITMIME")
			} else {
				reply("250-fake\r\n250 8BITMIME")
			}
		case command == "STARTTLS":
			reply("220 go ahead")
			conn = tls.Server(conn, s.tlsConfig)
			reader = bufio.NewReader(conn)
			s.startTLS = false
		case strings.HasPrefix(command, "MAIL FROM:"):
			current.from = extractAddress(line)
			reply("250 ok")
		case strings.HasPrefix(command, "RCPT TO:"):
			current.to = extractAddress(line)
			reply("250 ok")
		case command == "DATA":
			reply("354 go")
			var data strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil || dataLine == ".\r\n" {
					break
				}
				data.WriteString(dataLine)
			}
			current.data = data.String()
			s.mu.Lock()
			s.received = append(s.received, current)
			s.mu.Unlock()
			current = receivedMail{}
			reply("250 queued")
		case command == "QUIT":
			reply("221 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

func extractAddress(line string) string {
	start, end := strings.Index(line, "<"), strings.Index(line, ">")
	if start < 0 || end < start {
		return ""
	}
	return line[start+1 : end]
}

func (s *fakeSMTP) messages() []receivedMail {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]receivedMail(nil), s.received...)
}

func selfSignedCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		DNSNames:              []string{"localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(parsed)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pool
}

func TestSMTPMailerDeliversInEveryTLSMode(t *testing.T) {
	for _, mode := range []config.TLSMode{config.TLSModeNone, config.TLSModeStartTLS, config.TLSModeImplicit} {
		t.Run(string(mode), func(t *testing.T) {
			server, smtpConfig, clientTLS := newFakeSMTP(t, mode)
			mailer := NewSMTPMailer(smtpConfig, testSender, clock.NewFake(testNow))
			mailer.tlsConfig = clientTLS

			if err := mailer.Send(context.Background(), testMessage()); err != nil {
				t.Fatalf("Send : %v", err)
			}

			got := server.messages()
			if len(got) != 1 {
				t.Fatalf("mails reçus = %d", len(got))
			}
			if got[0].from != "no-reply@heros-sos.example" || got[0].to != "steve@avengers.example" {
				t.Errorf("enveloppe = %+v", got[0])
			}
			if !strings.Contains(got[0].data, "Subject: ") || !strings.Contains(got[0].data, "multipart/alternative") {
				t.Errorf("contenu inattendu :\n%s", got[0].data)
			}
		})
	}
}

func TestSMTPMailerRejectsUntrustedCertificate(t *testing.T) {
	_, smtpConfig, _ := newFakeSMTP(t, config.TLSModeImplicit)
	mailer := NewSMTPMailer(smtpConfig, testSender, clock.NewFake(testNow)) // certificat inconnu du système

	if err := mailer.Send(context.Background(), testMessage()); err == nil {
		t.Fatal("un certificat auto-signé non approuvé doit faire échouer l'envoi")
	}
}

func TestSMTPMailerFailsFastWhenServerIsDown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	mailer := NewSMTPMailer(config.SMTP{Host: "127.0.0.1", Port: port, TLSMode: config.TLSModeNone, Timeout: time.Second},
		testSender, clock.NewFake(testNow))
	if err := mailer.Send(context.Background(), testMessage()); err == nil {
		t.Fatal("l'envoi vers un serveur arrêté doit échouer")
	}
}

func TestSMTPMailerHonoursContextCancellation(t *testing.T) {
	// Serveur qui accepte la connexion mais ne répond jamais.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	mailer := NewSMTPMailer(config.SMTP{
		Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port, TLSMode: config.TLSModeNone, Timeout: 10 * time.Second,
	}, testSender, clock.NewFake(testNow))

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := mailer.Send(ctx, testMessage()); err == nil {
		t.Fatal("l'envoi vers un serveur muet doit échouer")
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("l'envoi a ignoré le délai du contexte (%v)", elapsed)
	}
}

// ---- File d'envoi asynchrone ------------------------------------------------------------------

func queueConfig() config.Mail {
	return config.Mail{Workers: 3, QueueSize: 16, MaxAttempts: 3, RetryDelay: time.Millisecond, SendTimeout: time.Second}
}

func closeQueue(t *testing.T, queue *Queue) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := queue.Close(ctx); err != nil {
		t.Fatalf("Close : %v", err)
	}
}

func TestQueueDeliversEverythingBeforeClosing(t *testing.T) {
	recorder := NewRecorder()
	queue := NewQueue(recorder, queueConfig(), discardLogger())

	const count = 12
	for range count {
		if err := queue.Send(context.Background(), testMessage()); err != nil {
			t.Fatal(err)
		}
	}
	closeQueue(t, queue)

	if got := len(recorder.Messages()); got != count {
		t.Fatalf("mails envoyés = %d, attendu %d : la fermeture doit vider la file", got, count)
	}
}

// flakyMailer échoue failures fois puis réussit.
type flakyMailer struct {
	failures atomic.Int32
	calls    atomic.Int32
	sent     atomic.Int32
}

func (m *flakyMailer) Send(context.Context, Message) error {
	m.calls.Add(1)
	if m.failures.Add(-1) >= 0 {
		return errors.New("serveur indisponible")
	}
	m.sent.Add(1)
	return nil
}

func TestQueueRetriesThenSucceeds(t *testing.T) {
	mailer := &flakyMailer{}
	mailer.failures.Store(2)
	queue := NewQueue(mailer, queueConfig(), discardLogger())

	if err := queue.Send(context.Background(), testMessage()); err != nil {
		t.Fatal(err)
	}
	closeQueue(t, queue)

	if mailer.sent.Load() != 1 || mailer.calls.Load() != 3 {
		t.Fatalf("envois réussis = %d, tentatives = %d ; attendu 1 et 3", mailer.sent.Load(), mailer.calls.Load())
	}
}

func TestQueueGivesUpAfterMaxAttempts(t *testing.T) {
	mailer := &flakyMailer{}
	mailer.failures.Store(1000)
	queue := NewQueue(mailer, queueConfig(), discardLogger())

	_ = queue.Send(context.Background(), testMessage())
	closeQueue(t, queue)

	if mailer.sent.Load() != 0 || mailer.calls.Load() != 3 {
		t.Fatalf("envois réussis = %d, tentatives = %d ; attendu 0 et 3", mailer.sent.Load(), mailer.calls.Load())
	}
}

func TestQueueDoesNotRetryInvalidMessages(t *testing.T) {
	queue := NewQueue(NewRecorder(), queueConfig(), discardLogger())
	defer closeQueue(t, queue)

	bad := testMessage()
	bad.To = "pas-un-email"
	if err := queue.Send(context.Background(), bad); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("un message invalide doit être refusé immédiatement : %v", err)
	}
}

// blockingMailer reste bloqué jusqu'à release (ou l'annulation du contexte).
type blockingMailer struct {
	started chan struct{}
	release chan struct{}
}

func (m *blockingMailer) Send(ctx context.Context, _ Message) error {
	m.started <- struct{}{}
	select {
	case <-m.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestQueueReportsSaturationAndClosure(t *testing.T) {
	mailer := &blockingMailer{started: make(chan struct{}, 10), release: make(chan struct{})}
	cfg := queueConfig()
	cfg.Workers, cfg.QueueSize = 1, 2
	queue := NewQueue(mailer, cfg, discardLogger())

	// Le travailleur prend le premier mail et se bloque ; deux autres remplissent la file.
	_ = queue.Send(context.Background(), testMessage())
	<-mailer.started
	for range 2 {
		if err := queue.Send(context.Background(), testMessage()); err != nil {
			t.Fatalf("la file n'est pas encore pleine : %v", err)
		}
	}
	if err := queue.Send(context.Background(), testMessage()); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("erreur = %v, attendu ErrQueueFull", err)
	}

	close(mailer.release)
	closeQueue(t, queue)
	if err := queue.Send(context.Background(), testMessage()); !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("erreur = %v, attendu ErrQueueClosed", err)
	}
}

func TestQueueCloseDeadlineAbortsInFlightSends(t *testing.T) {
	mailer := &blockingMailer{started: make(chan struct{}, 10), release: make(chan struct{})}
	cfg := queueConfig()
	cfg.Workers = 1
	queue := NewQueue(mailer, cfg, discardLogger())
	_ = queue.Send(context.Background(), testMessage())
	<-mailer.started

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := queue.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close = %v, attendu DeadlineExceeded : l'envoi bloqué doit être interrompu", err)
	}
}

func TestQueueCloseIsIdempotent(t *testing.T) {
	queue := NewQueue(NewRecorder(), queueConfig(), discardLogger())
	closeQueue(t, queue)
	closeQueue(t, queue)
}

func TestQueueConcurrentProducersAndClose(t *testing.T) {
	recorder := NewRecorder()
	cfg := queueConfig()
	cfg.QueueSize = 4096
	queue := NewQueue(recorder, cfg, discardLogger())

	var accepted atomic.Int64
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				switch err := queue.Send(context.Background(), testMessage()); {
				case err == nil:
					accepted.Add(1)
				case errors.Is(err, ErrQueueClosed), errors.Is(err, ErrQueueFull):
				default:
					t.Errorf("erreur inattendue : %v", err)
				}
			}
		}()
	}
	// Fermeture au milieu des envois : aucune panique « send on closed channel » possible.
	time.Sleep(time.Millisecond)
	closeQueue(t, queue)
	wg.Wait()

	if got := int64(len(recorder.Messages())); got != accepted.Load() {
		t.Fatalf("mails délivrés = %d, mails acceptés = %d : un mail accepté ne doit jamais être perdu", got, accepted.Load())
	}
}

// ---- Service et modèles -----------------------------------------------------------------------

func TestServiceRendersActivationAndPasswordResetMails(t *testing.T) {
	recorder := NewRecorder()
	service, err := NewService(recorder, "Heros-SOS", config.Tokens{ActivationTTL: 24 * time.Hour, PasswordResetTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	const link = "https://heros-sos.example/activation?token=abc123"

	if err := service.SendActivation(context.Background(), "steve@avengers.example", "Steve", link); err != nil {
		t.Fatal(err)
	}
	if err := service.SendPasswordReset(context.Background(), "steve@avengers.example", "Steve", link); err != nil {
		t.Fatal(err)
	}

	messages := recorder.Messages()
	if len(messages) != 2 {
		t.Fatalf("mails = %d", len(messages))
	}
	tests := []struct {
		message     Message
		wantSubject string
		wantValid   string
	}{
		{messages[0], "Activez votre compte Heros-SOS", "24 heures"},
		{messages[1], "Réinitialisation de votre mot de passe Heros-SOS", "1 heure"},
	}
	for _, test := range tests {
		if test.message.Subject != test.wantSubject || test.message.To != "steve@avengers.example" {
			t.Errorf("sujet = %q, destinataire = %q", test.message.Subject, test.message.To)
		}
		for name, content := range map[string]string{"texte": test.message.Text, "HTML": test.message.HTML} {
			for _, want := range []string{"Steve", link, test.wantValid} {
				if !strings.Contains(content, want) {
					t.Errorf("%s / %s : %q absent", test.wantSubject, name, want)
				}
			}
		}
	}
}

func TestServiceRendersHeroReviewRequest(t *testing.T) {
	recorder := NewRecorder()
	service, err := NewService(recorder, "Heros-SOS", config.Tokens{})
	if err != nil {
		t.Fatal(err)
	}
	const link = "https://heros-sos.example/profil/interventions?heroId=hero-1&interventionId=intervention-1"
	if err := service.SendReviewRequest(context.Background(), "steve@avengers.example", link); err != nil {
		t.Fatal(err)
	}
	message, ok := recorder.Last()
	if !ok || message.To != "steve@avengers.example" || message.Subject != "Évaluez l'intervention de Heros-SOS" {
		t.Fatalf("message reçu = %+v, trouvé=%v", message, ok)
	}
	if !strings.Contains(message.Text, link) || !strings.Contains(message.HTML, "Donner mon avis") || !strings.Contains(message.HTML, "heroId=hero-1") {
		t.Errorf("le lien de notation manque du message : %+v", message)
	}
}

func TestServiceEscapesUserControlledValuesInHTML(t *testing.T) {
	recorder := NewRecorder()
	service, _ := NewService(recorder, "Heros-SOS", config.Tokens{ActivationTTL: time.Hour, PasswordResetTTL: time.Hour})

	_ = service.SendActivation(context.Background(), "a@b.co", `<script>alert("xss")</script>`, "https://heros-sos.example/activation?token=a&x=1")

	message, _ := recorder.Last()
	if strings.Contains(message.HTML, "<script>") {
		t.Errorf("le prénom doit être échappé dans le HTML :\n%s", message.HTML)
	}
	if !strings.Contains(message.HTML, "&lt;script&gt;") {
		t.Errorf("prénom échappé absent :\n%s", message.HTML)
	}
}

func TestServiceRendersContactMessageForTheTeam(t *testing.T) {
	recorder := NewRecorder()
	service, _ := NewService(recorder, "Heros-SOS", config.Tokens{})

	err := service.SendContactMessage(context.Background(), "bureau@heros-sos.example", ContactMessage{
		SenderName: `Eve <script>alert(1)</script>`, SenderEmail: "eve@example.org",
		Subject: "Problème avec mon compte", Body: "Bonjour,\nje n'arrive plus à me connecter. <b>Merci</b>",
	})
	if err != nil {
		t.Fatal(err)
	}
	message, _ := recorder.Last()

	if message.To != "bureau@heros-sos.example" || message.ReplyTo != "eve@example.org" {
		t.Errorf("destinataire = %q, réponse = %q", message.To, message.ReplyTo)
	}
	if message.Subject != "[Heros-SOS] Message de contact : Problème avec mon compte" {
		t.Errorf("sujet = %q", message.Subject)
	}
	// Le nom et le texte, saisis par le visiteur, restent hors du sujet.
	if strings.Contains(message.Subject, "Eve") || strings.Contains(message.Subject, "connecter") {
		t.Errorf("le sujet ne doit pas contenir de saisie libre : %q", message.Subject)
	}
	if !strings.Contains(message.Text, "je n'arrive plus à me connecter") || !strings.Contains(message.Text, "eve@example.org") {
		t.Errorf("texte = %q", message.Text)
	}
	for _, injected := range []string{"<script>", "<b>"} {
		if strings.Contains(message.HTML, injected) {
			t.Errorf("%s doit être échappé dans le HTML :\n%s", injected, message.HTML)
		}
	}
	if !strings.Contains(message.HTML, "&lt;b&gt;Merci&lt;/b&gt;") {
		t.Errorf("le corps échappé manque du HTML :\n%s", message.HTML)
	}
}

func TestServiceRendersNewsletterConfirmation(t *testing.T) {
	recorder := NewRecorder()
	service, _ := NewService(recorder, "Heros-SOS", config.Tokens{NewsletterConfirmTTL: 48 * time.Hour})

	const link = "https://heros-sos.example/newsletter/confirmation?token=abc"
	if err := service.SendNewsletterConfirmation(context.Background(), "eve@example.org", link); err != nil {
		t.Fatal(err)
	}
	message, _ := recorder.Last()
	if message.To != "eve@example.org" || message.ReplyTo != "" || message.Subject != "Confirmez votre abonnement à la newsletter Heros-SOS" {
		t.Errorf("message = %+v", message)
	}
	for name, content := range map[string]string{"texte": message.Text, "HTML": message.HTML} {
		if !strings.Contains(content, link) || !strings.Contains(content, "48 heures") {
			t.Errorf("%s : le lien ou la durée de validité manque :\n%s", name, content)
		}
	}
}

func TestServicePropagatesMailerFailure(t *testing.T) {
	recorder := NewRecorder()
	recorder.FailWith(ErrQueueFull)
	service, _ := NewService(recorder, "Heros-SOS", config.Tokens{ActivationTTL: time.Hour, PasswordResetTTL: time.Hour})

	if err := service.SendActivation(context.Background(), "a@b.co", "Steve", "https://x.example"); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("erreur = %v", err)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := map[time.Duration]string{
		time.Hour:        "1 heure",
		24 * time.Hour:   "24 heures",
		30 * time.Minute: "30 minutes",
		time.Minute:      "1 minute",
		90 * time.Minute: "90 minutes",
	}
	for duration, want := range tests {
		if got := formatDuration(duration); got != want {
			t.Errorf("formatDuration(%v) = %q, attendu %q", duration, got, want)
		}
	}
}

func TestRecorderSimulatesFailureAndRecovery(t *testing.T) {
	recorder := NewRecorder()
	recorder.FailWith(errors.New("panne"))
	if err := recorder.Send(context.Background(), testMessage()); err == nil {
		t.Fatal("la panne simulée doit faire échouer l'envoi")
	}
	recorder.FailWith(nil)
	if err := recorder.Send(context.Background(), testMessage()); err != nil {
		t.Fatal(err)
	}
	if _, ok := NewRecorder().Last(); ok {
		t.Error("Last sur un enregistreur vide doit retourner false")
	}
}

func TestLogMailerRejectsInvalidMessages(t *testing.T) {
	mailer := NewLogMailer(discardLogger())
	if err := mailer.Send(context.Background(), testMessage()); err != nil {
		t.Fatal(err)
	}
	if err := mailer.Send(context.Background(), Message{To: "x"}); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("erreur = %v", err)
	}
}

func TestServiceRendersEmailChangeMails(t *testing.T) {
	recorder := NewRecorder()
	service, _ := NewService(recorder, "Heros-SOS", config.Tokens{EmailChangeConfirmTTL: 48 * time.Hour})
	ctx := context.Background()

	const link = "https://heros-sos.example/changement-email/confirmation?token=abc"
	if err := service.SendEmailChangeConfirmation(ctx, "nouvelle@example.org", "Steve", link); err != nil {
		t.Fatal(err)
	}
	if err := service.SendEmailChangeApproved(ctx, "nouvelle@example.org", "Steve"); err != nil {
		t.Fatal(err)
	}
	if err := service.SendEmailChangeNotice(ctx, "ancienne@example.org", "Steve", "nouvelle@example.org"); err != nil {
		t.Fatal(err)
	}
	if err := service.SendEmailChangeRejected(ctx, "ancienne@example.org", "Steve"); err != nil {
		t.Fatal(err)
	}

	sent := recorder.Messages()
	if len(sent) != 4 {
		t.Fatalf("%d mails, attendu 4", len(sent))
	}
	tests := []struct {
		to       string
		subject  string
		contains []string
	}{
		{"nouvelle@example.org", "Confirmez votre nouvelle adresse email Heros-SOS", []string{link, "48 heures", "Steve"}},
		{"nouvelle@example.org", "Votre nouvelle adresse email est active sur Heros-SOS", []string{"Steve", "adresse de connexion"}},
		{"ancienne@example.org", "L'adresse email de votre compte Heros-SOS a été remplacée", []string{"nouvelle@example.org", "contactez-nous"}},
		{"ancienne@example.org", "Votre demande de changement d'adresse email a été refusée", []string{"Steve", "ne change pas"}},
	}
	for i, want := range tests {
		message := sent[i]
		if message.To != want.to || message.Subject != want.subject {
			t.Errorf("mail %d : %s / %q, attendu %s / %q", i, message.To, message.Subject, want.to, want.subject)
		}
		for name, content := range map[string]string{"texte": message.Text, "HTML": message.HTML} {
			for _, expected := range want.contains {
				if !strings.Contains(content, expected) {
					t.Errorf("mail %d (%s) : %q manque :\n%s", i, name, expected, content)
				}
			}
		}
	}
}
