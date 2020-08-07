package external

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/mailgun/mailgun-go/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type TemplateInfo struct {
	name    string
	subject string
	inlines []string
	tag     string
}

var (
	templateInfo = map[EmailType]TemplateInfo{
		EmailType_Waitlist: TemplateInfo{
			name:    "waitlist",
			subject: "Your waitlist code",
			inlines: []string{
				"facebook_square_grey.png",
				"ggwp_logo.png",
				"influencer.png",
				"instagram_square_grey.png",
				"linkedin_square_grey.png",
				"twitter_square_grey.png",
			},
		},
	}
	sender = "noreply@ggwpacademy.com"
)

const (
	templatePath = "./app/mailer/templates"
)

type Mailer struct {
	mg  *mailgun.MailgunImpl
	log *logrus.Entry
}

func NewMailer(log *logrus.Entry) *Mailer {
	return &Mailer{
		mg: mailgun.NewMailgun(
			os.Getenv("MAIL_GUN_DOMAIN"),
			os.Getenv("MAIL_GUN_API_KEY"),
		),
		log: log,
	}
}

func (m *Mailer) SendForgotPassword(
	ctx context.Context,
	recipient,
	token string,
) error {
	subject := "Forgot Password"
	body := fmt.Sprintf("Your password reset token is: %q", token)

	// The message object allows you to add attachments and Bcc recipients
	message := m.mg.NewMessage(sender, subject, body, recipient)
	if err := m.sendEmail(message); err != nil {
		return errors.Wrapf(err, "sending password reset to %s", recipient)
	}
	return nil
}

func (m *Mailer) SendWaitlistEmail(
	ctx context.Context,
	recipient,
	waitlistCode string,
) error {
	subject := "Your Waitilst Code"

	text, html, err := GenerateEmail(
		EmailType_Waitlist,
		WaitlistEmailVars{
			WaitlistCode: waitlistCode,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "getting templates for %s", EmailType_Waitlist)
	}

	// The message object allows you to add attachments and Bcc recipients
	message := m.mg.NewMessage(sender, subject, text, recipient)
	message.SetHtml(html)

	info, ok := templateInfo[EmailType_Waitlist]
	if !ok {
		return fmt.Errorf("no template info for email type: %s", EmailType_Waitlist)
	}
	for _, i := range info.inlines {
		message.AddInline(fmt.Sprintf("%s/%s", templatePath, i))
	}
	message.AddTag(EmailType_Waitlist.String())

	if err := m.sendEmail(message); err != nil {
		return errors.Wrapf(err, "sending waitlist email to %s", recipient)
	}
	return nil
}

func GenerateEmail(t EmailType, vars interface{}) (
	text,
	html string,
	err error,
) {
	info, ok := templateInfo[t]
	if !ok {
		return "", "", fmt.Errorf("no template info for email type: %s", t)
	}
	textTemplateName := fmt.Sprintf("%s/%s.txt", templatePath, info.name)
	htmlTemplateName := fmt.Sprintf("%s/%s.html", templatePath, info.name)

	textTemplate, err := template.ParseFiles(textTemplateName)
	if err != nil {
		return "", "", errors.Wrapf(err, "parsing text template: %s", textTemplateName)
	}

	htmlTemplate, err := template.ParseFiles(htmlTemplateName)
	if err != nil {
		return "", "", errors.Wrapf(err, "parsing html template: %s", htmlTemplateName)
	}

	textBuf := &bytes.Buffer{}
	if err := textTemplate.Execute(textBuf, vars); err != nil {
		return "", "", errors.Wrapf(err, "executing text template %s", t)
	}
	text = textBuf.String()

	htmlBuf := &bytes.Buffer{}
	if err := htmlTemplate.Execute(htmlBuf, vars); err != nil {
		return "", "", errors.Wrapf(err, "executing html template %s", t)
	}
	html = htmlBuf.String()
	return
}

func (m *Mailer) sendEmail(message *mailgun.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// add tracking
	message.SetTracking(true)
	message.SetTrackingClicks(true)
	message.SetTrackingOpens(true)

	_, _, err := m.mg.Send(ctx, message)
	if err != nil {
		return errors.Wrap(err, "sending email")
	}
	return nil
}

func (m *Mailer) SendEmail(
	ctx context.Context,
	e Email,
) error {
	text, html, err := GenerateEmail(
		e.Type,
		e.TemplateVars,
	)
	if err != nil {
		return errors.Wrapf(err, "getting templates for %s", EmailType_Waitlist)
	}

	info, ok := templateInfo[EmailType_Waitlist]
	if !ok {
		return fmt.Errorf("no template info for email type: %s", EmailType_Waitlist)
	}

	// The message object allows you to add attachments and Bcc recipients
	message := m.mg.NewMessage(sender, info.subject, text, e.EmailAddress)
	message.SetHtml(html)

	for _, i := range info.inlines {
		message.AddInline(fmt.Sprintf("%s/%s", templatePath, i))
	}
	message.AddTag(EmailType_Waitlist.String())

	if err := m.sendEmail(message); err != nil {
		return errors.Wrapf(err, "sending waitlist email to %s", e.EmailAddress)
	}
	return nil
}
