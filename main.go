package main

import (
	"crypto/tls"
	"encoding/json"
	"github.com/SourceForgery/tachikoma-bridge/build/generated/github.com/SourceForgery/tachikoma"
	"github.com/SourceForgery/tachikoma-bridge/nodemailer"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"io"
	"net/http"
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
	APIKey          string `short:"a" long:"apikey" description:"ApiKey e.g. example.com:mysupersecretpassword"`
	Uri             string `short:"u" long:"uri" description:"URI to Tachikoma"`
	ListeningPort   string `short:"p" long:"port" default:"3100" description:"Listening Port"`
	WebhookUri      string `short:"w" long:"webhook" description:"URI to Parcelvoy webhook"`
}

func parseOptions() {
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
	nodemailer.Logger = logger

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

	if opts.WebhookUri == "" {
		logger.Fatal().Msg("Please provide webhook URI though param --webhook")
	}

	if opts.Uri == "" {
		logger.Fatal().Msg("Please provide uri through param --uri")
	}
	if opts.APIKey == "" {
		opts.APIKey = os.Getenv("TACHIKOMA_AUTH")
		if opts.APIKey == "" {
			logger.Fatal().Msg("Please provide with an API-Key either through params --apikey or env TACHIKOMA_AUTH")
		}
	}
}

func createGrpcConn() (conn *grpc.ClientConn, ctx context.Context) {
	uri, err := url.Parse(opts.Uri)
	if err != nil {
		logger.Fatal().Err(err).Msgf("Not a valid URI: %s", opts.Uri)
	}
	cert, err := tls.LoadX509KeyPair(opts.CertificatePath, opts.KeyPath)
	if err != nil {
		logger.Fatal().Err(err).Msgf("failed to load client certificates: '%s' and '%s'", opts.CertificatePath, opts.KeyPath)
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
	})
	port := "443"
	if uri.Port() != "" {
		port = uri.Port()
	}
	conn, err = grpc.Dial(uri.Host+":"+port, grpc.WithTransportCredentials(creds))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to dial gRPC server")
	}
	ctx = metadata.AppendToOutgoingContext(context.Background(), "x-apitoken", opts.APIKey)
	return
}

func main() {
	parseOptions()
	conn, ctx := createGrpcConn()
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	go serve(conn, ctx)

	err := streamNotifications(conn, ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to receive notification")
	}
}

func streamNotifications(conn *grpc.ClientConn, ctx context.Context) (err error) {
	webhookUrl, err := url.Parse(opts.WebhookUri)
	if err != nil {
		return
	}
	client := tachikoma.NewDeliveryNotificationServiceClient(conn)
	streamClient, err := client.NotificationStreamWithKeepAlive(
		ctx,
		&tachikoma.NotificationStreamParameters{
			IncludeTrackingData: true,
			IncludeSubject:      true,
			IncludeMetricsData:  true,
		},
	)
	if err != nil {
		return
	}
	for {
		var notification *tachikoma.EmailNotificationOrKeepAlive
		notification, err = streamClient.Recv()
		if err == io.EOF {
			return
		}
		if notification == nil {
			continue
		}
		if err != nil {
			return
		}
		if notification.GetEmailNotification() != nil {
			nodemailer.SendEvent(notification.GetEmailNotification(), webhookUrl)
		}
	}
}

func parseAndSend(conn *grpc.ClientConn, ctx context.Context, body io.ReadCloser) (err error) {
	bytes, err := io.ReadAll(body)
	if err != nil {
		return
	}
	//logger.Info().Msg(string(bytes))
	var email nodemailer.NodeMailerEmail
	err = json.Unmarshal(bytes, &email)
	if err != nil {
		return
	}
	logger.Debug().Any("msg", email).Msg(string(bytes))
	_, err = nodemailer.SendMail(conn, ctx, email)
	if err != nil {
		return
	}
	return
}

func serve(conn *grpc.ClientConn, ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/tachikoma/sendEmail", func(writer http.ResponseWriter, request *http.Request) {
		body := request.Body
		defer func(body io.ReadCloser) {
			_ = body.Close()
		}(body)
		err := parseAndSend(conn, ctx, body)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to parse send email")
			writer.WriteHeader(400)
		} else {
			writer.WriteHeader(204)
		}

		_, err = writer.Write([]byte{})
		if err != nil {
			logger.Error().Err(err).Msg("write failed")
		}
	})

	var addr = ":" + opts.ListeningPort
	logger.Info().Msgf("Listening to %s", addr)
	err := http.ListenAndServe(addr, mux)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to start http server")
	}
}

//go:generate go run buildscripts/gen.go
//go:generate protoc --go_out=build/generated --go-grpc_out=build/generated -I=./build/protobuf build/protobuf/com/sourceforgery/tachikoma/grpc/frontend/maildelivery/maildelivery.proto build/protobuf/com/sourceforgery/tachikoma/grpc/frontend/tracking/mailtracking.proto build/protobuf/com/sourceforgery/tachikoma/grpc/frontend/common.proto
