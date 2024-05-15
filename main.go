package main

import (
	"crypto/tls"
	"github.com/SourceForgery/tachikoma-bridge/build/generated/github.com/SourceForgery/tachikoma"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"io"
	"net/url"
	"os"
	"time"
)

var logger zerolog.Logger

var opts struct {
	// Slice of bool will append 'true' each time the option
	// is encountered (can be set multiple times, like -vvv)
	Verbose         []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	Quiet           bool   `short:"q" long:"quiet" description:"Be very quiet"`
	LoggingFormat   string `short:"l" long:"logging" choice:"coloured" choice:"plain" choice:"json" default:"coloured" description:"Log output format"`
	CertificatePath string `short:"c" long:"certificate" description:"Path to certificate"`
	KeyPath         string `short:"k" long:"key" description:"Path to Key"`
	APIKey          string `short:"a" long:"apikey" description:"ApiKey"`
	Uri             string `short:"u" long:"uri" description:"URI to Tachikoma"`
	ListeningPort   string `short:"p" long:"port" default:"3100" description:"Listening Port"`
	WebhookUri      string `short:"w" long:"webhook" description:"URI to Parcelvoy webhook"`
}

func initalization() {
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		log.Err(err)
		os.Exit(1)
	}

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	switch opts.LoggingFormat {
	case "json":
		logger = zerolog.New(os.Stdout)
		break
	case "coloured":
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
		break
	case "plain":
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339, NoColor: true})
		break
	default:
		logger.Panic().Msgf("What the f is %s", opts.LoggingFormat)
	}
	logger = logger.With().Timestamp().Logger()

	switch len(opts.Verbose) {
	case 0:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		break
	case 1:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		break
	case 2:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
		break
	}

	if opts.Quiet {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
}

func main() {
	initalization()
	url, err := url.Parse(opts.Uri)
	if err != nil {
		log.Fatal().Err(err).Msgf("Not a valid URI: %s", opts.Uri)
	}

	// Load the client certificates from disk
	cert, err := tls.LoadX509KeyPair(opts.CertificatePath, opts.KeyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load client certificate")
	}

	// Load the CA certificate
	//caCert, err := ioutil.ReadFile("ca.crt")
	//if err != nil {
	//	log.Fatalf("failed to read CA certificate: %v", err)
	//}
	//caCertPool := x509.NewCertPool()
	//if !caCertPool.AppendCertsFromPEM(caCert) {
	//	log.Fatalf("failed to append CA certificate to pool")
	//}

	// Create the transport credentials
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		//RootCAs:      caCertPool,
	})
	port := "443"
	if url.Port() != "" {
		port = url.Port()
	}
	// Create a new gRPC client with the credentials
	conn, err := grpc.Dial(url.Host+":"+port, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial gRPC server")
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-apitoken", opts.APIKey)

	//_, err = sendMail(conn, ctx)
	//if err != nil {
	//	log.Fatal().Err(err).Msg("sendMail function failed, go shoot yourself")
	//}

	client := tachikoma.NewDeliveryNotificationServiceClient(conn)
	streamClient, err := client.NotificationStreamWithKeepAlive(ctx, &tachikoma.NotificationStreamParameters{
		IncludeTrackingData: true,
		IncludeSubject:      true,
		IncludeMetricsData:  true,
		Tags:                nil,
	})
	for {
		notification, err := streamClient.Recv()
		if err == io.EOF {
			// End of stream
			break
		}
		if err != nil {
			log.Fatal().Err(err).Msg("failed to receive notification")
		}
		if notification.GetEmailNotification() != nil {
			handleNotification(notification.GetEmailNotification())
		}
	}
}

func handleNotification(notification *tachikoma.EmailNotification) {
	log.Info().Msgf("got notification with folloing data: %s", notification.String())
}

func sendMail(conn *grpc.ClientConn, ctx context.Context) (response *tachikoma.EmailQueueStatus, err error) {
	client := tachikoma.NewMailDeliveryServiceClient(conn)
	namedEmailAdress := tachikoma.NamedEmailAddress{Name: "hej", Email: "victor.lindell@youcruit.com"}
	emailRecipent := tachikoma.EmailRecipient{NamedEmail: &namedEmailAdress}
	recipents := []*tachikoma.EmailRecipient{&emailRecipent}
	staticBody := tachikoma.StaticBody{
		HtmlBody: "this is an <i>html</i> <b>body</b>",
		Subject:  "tachikoma bridge works: " + time.Now().String(),
	}
	caos := tachikoma.OutgoingEmail_Static{
		Static: &staticBody,
	}
	outgoingEmail := tachikoma.OutgoingEmail{
		Recipients: recipents,
		Bcc:        nil,
		From: &tachikoma.NamedEmailAddress{
			Email: "victor.lindell@staging.youcruit.com",
			Name:  "Victor Lindell",
		},
		ReplyTo: &tachikoma.EmailAddress{
			Email: "victor.lindell@staging.youcruit.com",
		},
		Body:    &caos,
		Headers: nil,
		TrackingData: &tachikoma.TrackingData{
			TrackingDomain: "",
			Tags:           nil,
			Metadata:       nil,
		},
		SendAt: &timestamp.Timestamp{
			Seconds: 0,
			Nanos:   0,
		},
		SigningDomain:          "",
		Attachments:            nil,
		RelatedAttachments:     nil,
		UnsubscribeRedirectUri: "",
		InlineCss:              false,
	}

	resp, err := client.SendEmail(ctx, &outgoingEmail)
	if err != nil {
		return nil, errors.Wrap(err, "client sendmail failed")
	}
	response, err = resp.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "getting sendmail response failed")
	}
	log.Info().Msgf("Response: %s", response)
	return
}

//go:generate go run buildscripts/gen.go
//go:generate protoc --go_out=build/generated --go-grpc_out=build/generated -I=./build/protobuf build/protobuf/com/sourceforgery/tachikoma/grpc/frontend/maildelivery/maildelivery.proto build/protobuf/com/sourceforgery/tachikoma/grpc/frontend/tracking/mailtracking.proto build/protobuf/com/sourceforgery/tachikoma/grpc/frontend/common.proto
