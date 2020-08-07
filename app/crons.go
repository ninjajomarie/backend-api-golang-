package external

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	MISSING_REFERRAL_CODES_SLEEP = 1 * time.Minute
	MISSING_WAITLIST_CODES_SLEEP = 1 * time.Minute
	QUEUE_WAITLIST_EMAILS_SLEEP  = 5 * time.Minute

	SEND_EMAILS_SLEEP = 5 * time.Minute
)

func (e *External) RunCrons() {
	go e.processMissingReferralCodes()
	go e.processMissingWaitlistCodes()
	go e.queueWaitlistEmails()

	go e.sendEmails()
}

// create referral codes for users missing them (every 1 minute)
func (e *External) processMissingReferralCodes() {
	for {
		e.log.Info("starting to process users missing referral codes")
		userIDs, err := GetUsersIDsMissingReferralCodes(e.dao.ReadDB)
		if err != nil {
			e.log.WithError(err).Error("unable to get user ids missing referral codes")
		}

		e.log.WithField("count", len(userIDs)).Info("processing")
		for _, userID := range userIDs {
			_, err := GenerateReferralCode(e.dao.DB, userID)
			if err != nil {
				e.log.WithField("user_id", userID).WithError(err).Error("unable to generate referral code for user")
			}
		}
		e.log.WithField("count", len(userIDs)).Info("done processing users missing referral codes")

		time.Sleep(MISSING_REFERRAL_CODES_SLEEP)
	}
}

// create waitlist codes for users missing them (every 1 minute)
func (e *External) processMissingWaitlistCodes() {
	for {
		e.log.Info("starting to process users missing waitlist codes")
		emails, err := GetEmailsMissingWaitlistCodes(e.dao.ReadDB)
		if err != nil {
			e.log.WithError(err).Error("unable to get emails missing waitlist codes")
		}

		e.log.WithField("count", len(emails)).Info("processing")
		for _, email := range emails {
			if err := GenerateAndWriteWaitlistCode(e.dao.DB, email); err != nil {
				e.log.WithField("user_id", email).WithError(err).Error("unable to generate waitlist code for user")
			}
		}
		e.log.WithField("count", len(emails)).Info("done processing users missing waitlist codes")

		time.Sleep(MISSING_WAITLIST_CODES_SLEEP)
	}
}

// queue waitlist emails (every 1 minute)
func (e *External) queueWaitlistEmails() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		tx, err := e.dao.GetTx(ctx)
		if err != nil {
			e.log.Error("creating transaction")
			return
		}
		defer tx.Rollback()

		items, err := GetWaitlistItemsWithoutAQueuedEmail(tx)
		if err != nil {
			e.log.WithError(err).Error("getting waitlist items without a queued email")
			return
		}

		e.log.WithField("count", len(items)).Info("starting to process emails not queued up for the waitlist")
		for _, i := range items {
			if err := CreateEmail(
				tx,
				nil,
				i.EmailAddress,
				// templateInfo[EmailType_Waitlist]["name"],
				"waitlist",
				HStoreMap{
					"WaitlistCode": i.OwnerWaitlistCode,
				},
				EmailType_Waitlist,
				EmailStatus_Pending,
			); err != nil {
				e.log.
					WithField("email_address", i.EmailAddress).
					WithError(err).
					Error("creating email for waitlist user")
			}
		}

		if err := tx.Commit(); err != nil {
			e.log.Info("commiting changes")
			return
		}

		e.log.WithField("count", len(items)).Info("done processing emails for waitlist users not queued")
		time.Sleep(QUEUE_WAITLIST_EMAILS_SLEEP)
	}
}

// send emails (every 5 minute)
func (e *External) sendEmails() {
	for {
		items, err := GetPendingEmails(e.dao.ReadDB)
		if err != nil {
			e.log.WithError(err).Error("getting pending emails")
			return
		}

		e.log.WithField("count", len(items)).Info("starting to process unsent emails")
		m := NewMailer(e.log)
		for _, i := range items {
			l := e.log.
				WithFields(
					logrus.Fields{
						"email_address": i.EmailAddress,
						"type":          i.Type,
					},
				)
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			l.Info("sending")
			if err := m.SendEmail(ctx, *i); err != nil {
				l.WithError(err).Error("sending email for user")
				continue
			}
			l.Info("marking as sent")
			if err := MarkEmailAsSent(e.dao.DB, i.ID); err != nil {
				l.WithError(err).Error("marking email as sent")
			}
			l.Info("finished processing")
		}

		e.log.Info("done processing unsent emails")
		time.Sleep(SEND_EMAILS_SLEEP)
	}
}
