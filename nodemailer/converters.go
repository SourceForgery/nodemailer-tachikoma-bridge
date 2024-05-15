package nodemailer

import (
	"encoding/base64"
	"github.com/SourceForgery/tachikoma-bridge/build/generated/github.com/SourceForgery/tachikoma"
)

func (address NodeMailerAddress) ToTachikoma() *tachikoma.NamedEmailAddress {
	return &tachikoma.NamedEmailAddress{
		Email: address.Address,
		Name:  address.Name,
	}
}

func ToTachikomaNamedEmaiLAddresses(addresses []NodeMailerAddress) (namedAddresses []*tachikoma.NamedEmailAddress) {
	for _, address := range addresses {
		namedAddresses = append(namedAddresses, address.ToTachikoma())
	}
	return namedAddresses
}

func ToTachikomaEmaiLAddresses(addresses []string) (emailAddresses []*tachikoma.EmailAddress) {
	for _, address := range addresses {
		emailAddresses = append(emailAddresses, &tachikoma.EmailAddress{Email: address})
	}
	return emailAddresses
}

func ToTachikomaRecipients(addresses []NodeMailerAddress) (namedAddresses []*tachikoma.EmailRecipient) {
	for _, address := range addresses {
		namedAddresses = append(
			namedAddresses,
			&tachikoma.EmailRecipient{
				NamedEmail: address.ToTachikoma(),
			},
		)
	}
	return namedAddresses
}

func convertToTachikoma(email NodeMailerEmail) (tachikomaEmail tachikoma.OutgoingEmail, err error) {
	attachments := make([]*tachikoma.Attachment, len(email.Attachments))
	for i, attachment := range email.Attachments {
		var data []byte
		data, err = base64.StdEncoding.DecodeString(attachment.Base64Content)
		if err != nil {
			return
		}
		attachments[i] = &tachikoma.Attachment{
			Data:        data,
			ContentType: attachment.ContentType,
			FileName:    attachment.Filename,
		}
	}

	recipients := ToTachikomaRecipients(email.To)
	recipients = append(recipients, ToTachikomaRecipients(email.Cc)...)
	tachikomaEmail = tachikoma.OutgoingEmail{
		Recipients: recipients,
		Bcc:        ToTachikomaEmaiLAddresses(email.Bcc),
		From:       email.From.ToTachikoma(),
		ReplyTo: &tachikoma.EmailAddress{
			Email: email.ReplyTo,
		},

		Body: &tachikoma.OutgoingEmail_Static{
			Static: &tachikoma.StaticBody{
				HtmlBody: email.Html,
				Subject:  email.Text,
			},
		},
		TrackingData: &tachikoma.TrackingData{
			Metadata: map[string]string{CAMPAIGN_METADATA_KEY: email.CampaignId},
		},
		Attachments: attachments,
	}
	return
}
