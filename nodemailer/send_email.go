package nodemailer

import (
	"context"
	"github.com/SourceForgery/tachikoma-bridge/build/generated/github.com/SourceForgery/tachikoma"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func SendMail(conn *grpc.ClientConn, ctx context.Context, email NodeMailerEmail) (response *tachikoma.EmailQueueStatus, err error) {
	client := tachikoma.NewMailDeliveryServiceClient(conn)
	outgoingEmail, err := convertToTachikoma(email)
	if err != nil {
		return
	}

	resp, err := client.SendEmail(ctx, &outgoingEmail)
	if err != nil {
		return nil, errors.Wrap(err, "client sendmail failed")
	}
	response, err = resp.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "getting sendmail response failed")
	}
	log.Info().Msgf("Response: %s", response.String())
	return
}
