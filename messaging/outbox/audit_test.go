package outbox

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/yogayulanda/go-core/logger"
)

type auditLogRecorder struct{ events []logger.EventLog }

func (*auditLogRecorder) Info(context.Context, string, ...logger.Field)  {}
func (*auditLogRecorder) Error(context.Context, string, ...logger.Field) {}
func (*auditLogRecorder) Debug(context.Context, string, ...logger.Field) {}
func (*auditLogRecorder) Warn(context.Context, string, ...logger.Field)  {}
func (*auditLogRecorder) LogService(context.Context, logger.ServiceLog)  {}
func (*auditLogRecorder) LogDB(context.Context, logger.DBLog)            {}
func (l *auditLogRecorder) LogEvent(_ context.Context, event logger.EventLog) {
	l.events = append(l.events, event)
}
func (*auditLogRecorder) LogTransaction(context.Context, logger.TransactionLog) {}
func (l *auditLogRecorder) WithComponent(string) logger.Logger                  { return l }

func TestPublishAuditWritesCloudEventsHeadersAndNeverLogsPayload(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO outbox_events").WithArgs(
		"11111111-1111-4111-8111-111111111111", "audit-events", []byte("account-1"), []byte("payload-secret-value"), sqlmock.AnyArg(), "PENDING", sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(0, 1))
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	log := &auditLogRecorder{}
	publisher := NewPublisher(WithPublisherDriver("mysql")).WithLogger(log)
	const payloadSecret = "payload-secret-value"
	err = publisher.PublishAudit(context.Background(), tx, AuditEvent{
		Topic: "audit-events", SpecVersion: "1.0", ID: "11111111-1111-4111-8111-111111111111", Type: "account.changed.v1",
		Source: "/accounts", Time: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), Subject: "account-1", Sequence: "42", Data: []byte(payloadSecret),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(log.events) != 1 || log.events[0].Metadata["ce_sequence"] != "42" || log.events[0].Metadata["ce_id"] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("missing CloudEvents audit metadata: %#v", log.events)
	}
	encodedLog, err := json.Marshal(log.events)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encodedLog), payloadSecret) {
		t.Fatal("audit payload appeared in event log")
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditEventValidationRequiresSubjectSequenceAndType(t *testing.T) {
	base := AuditEvent{Topic: "audit", SpecVersion: "1.0", ID: "11111111-1111-4111-8111-111111111111", Type: "account.changed.v1", Source: "/accounts", Time: time.Now(), Subject: "account-1", Sequence: "1"}
	for _, mutate := range []func(*AuditEvent){
		func(event *AuditEvent) { event.Subject = "" },
		func(event *AuditEvent) { event.Sequence = "" },
		func(event *AuditEvent) { event.Type = "AccountChanged" },
		func(event *AuditEvent) { event.ID = "not-a-uuid" },
	} {
		event := base
		mutate(&event)
		if err := event.Validate(); err == nil {
			t.Errorf("expected invalid audit event to fail: %+v", event)
		}
	}
}
