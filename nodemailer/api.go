package nodemailer

const CAMPAIGN_METADATA_KEY = "campaign"

/**
 * Subset of the nodemailer api for easy integration.
 */

type NodeMailerAddress struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type NodeMailerAttachment struct {
	Base64Content string `json:"content"`
	Filename      string `json:"filename"`
	ContentType   string `json:"contentType"`
}

type NodeMailerEmail struct {
	From        NodeMailerAddress      `json:"from"`
	To          []NodeMailerAddress    `json:"to"`
	Cc          []NodeMailerAddress    `json:"cc"`
	Bcc         []string               `json:"bcc"`
	Subject     string                 `json:"subject"`
	Text        string                 `json:"text"`
	Html        string                 `json:"html"`
	Attachments []NodeMailerAttachment `json:"attachments"`
	ReplyTo     string                 `json:"replyTo"`
	Headers     map[string]string      `json:"headers"` // This is make-believe
	//InReplyTo   string                 `json:"inReplyTo"`
	//References  []string               `json:"references"`
}
