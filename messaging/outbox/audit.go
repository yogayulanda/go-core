package outbox

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
)

var auditTypePattern = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9_]+)+\.v[0-9]+$`)

type AuditEvent struct {
	Topic       string
	SpecVersion string
	ID          string
	Type        string
	Source      string
	Time        time.Time
	Subject     string
	Sequence    string
	Data        []byte
}

func (p *Publisher) WithLogger(log logger.Logger) *Publisher {
	if p != nil {
		p.log = log
	}
	return p
}

func (p *Publisher) PublishAudit(ctx context.Context, tx *sql.Tx, event AuditEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	headers := map[string]string{
		"ce_specversion": event.SpecVersion,
		"ce_id":          event.ID,
		"ce_type":        event.Type,
		"ce_source":      event.Source,
		"ce_time":        event.Time.UTC().Format(time.RFC3339Nano),
		"ce_subject":     event.Subject,
		"ce_sequence":    event.Sequence,
	}
	if err := p.PublishTx(ctx, tx, messaging.Message{Topic: event.Topic, Key: []byte(event.Subject), Payload: event.Data, Headers: headers, EventID: event.ID}); err != nil {
		return err
	}
	if p.log != nil {
		p.log.LogEvent(ctx, logger.EventLog{
			Category: "audit",
			Action:   event.Type,
			Outcome:  "success",
			Metadata: map[string]interface{}{"ce_id": event.ID, "ce_type": event.Type, "ce_subject": event.Subject, "ce_sequence": event.Sequence},
		})
	}
	return nil
}

func (event AuditEvent) Validate() error {
	if strings.TrimSpace(event.Topic) == "" || strings.TrimSpace(event.SpecVersion) == "" || strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.Type) == "" || strings.TrimSpace(event.Source) == "" || event.Time.IsZero() || strings.TrimSpace(event.Subject) == "" || strings.TrimSpace(event.Sequence) == "" {
		return fmt.Errorf("audit event requires topic and all CloudEvents attributes including ce_subject and ce_sequence")
	}
	if event.SpecVersion != "1.0" {
		return fmt.Errorf("unsupported ce_specversion %q", event.SpecVersion)
	}
	if _, err := uuid.Parse(event.ID); err != nil {
		return fmt.Errorf("ce_id must be a UUID: %w", err)
	}
	if !auditTypePattern.MatchString(event.Type) {
		return fmt.Errorf("invalid ce_type %q", event.Type)
	}
	return nil
}
