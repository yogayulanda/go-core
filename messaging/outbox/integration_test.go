//go:build integration

package outbox

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
)

type integrationPublisher struct{ events []messaging.Message }

func (p *integrationPublisher) Publish(_ context.Context, event messaging.Message) error {
	p.events = append(p.events, event)
	return nil
}
func (*integrationPublisher) Close() error { return nil }

type integrationLogger struct{}

func (integrationLogger) Info(context.Context, string, ...logger.Field)         {}
func (integrationLogger) Error(context.Context, string, ...logger.Field)        {}
func (integrationLogger) Debug(context.Context, string, ...logger.Field)        {}
func (integrationLogger) Warn(context.Context, string, ...logger.Field)         {}
func (integrationLogger) LogService(context.Context, logger.ServiceLog)         {}
func (integrationLogger) LogDB(context.Context, logger.DBLog)                   {}
func (integrationLogger) LogEvent(context.Context, logger.EventLog)             {}
func (integrationLogger) LogTransaction(context.Context, logger.TransactionLog) {}
func (integrationLogger) WithComponent(string) logger.Logger                    { return integrationLogger{} }

func TestOutboxIntegrationSQLServerAndMySQL(t *testing.T) {
	tests := []struct{ driver, env string }{{"sqlserver", "GO_CORE_TEST_SQLSERVER_DSN"}, {"mysql", "GO_CORE_TEST_MYSQL_DSN"}}
	for _, test := range tests {
		t.Run(test.driver, func(t *testing.T) {
			dsn := os.Getenv(test.env)
			if dsn == "" {
				t.Skipf("%s is not set; Docker is unavailable in this environment", test.env)
			}
			db, err := sql.Open(test.driver, dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			drop := "DROP TABLE IF EXISTS outbox_events"
			if test.driver == "sqlserver" {
				drop = "IF OBJECT_ID('dbo.outbox_events', 'U') IS NOT NULL DROP TABLE dbo.outbox_events"
			}
			if _, err := db.ExecContext(context.Background(), drop); err != nil {
				t.Fatalf("drop prior integration table: %v", err)
			}
			schemaPath := filepath.Join("testdata", "schema_"+test.driver+".sql")
			schema, err := os.ReadFile(schemaPath)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(context.Background(), string(schema)); err != nil {
				t.Fatalf("apply outbox schema: %v", err)
			}
			defer func() { _, _ = db.ExecContext(context.Background(), drop) }()
			tx, err := db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			publisher := NewPublisher(WithPublisherDriver(test.driver))
			if err := publisher.PublishTx(context.Background(), tx, messaging.Message{Topic: "integration.audit", Key: []byte("account-1"), Payload: []byte("payload")}); err != nil {
				_ = tx.Rollback()
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			kafkaPublisher := &integrationPublisher{}
			worker := NewWorkerWithOptions(db, kafkaPublisher, integrationLogger{}, WithWorkerDriver(test.driver))
			if err := worker.RunOnce(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(kafkaPublisher.events) != 1 {
				t.Fatalf("expected one published event, got %d", len(kafkaPublisher.events))
			}
			var status string
			query := "SELECT status FROM outbox_events WHERE event_id = ?"
			if test.driver == "sqlserver" {
				query = "SELECT status FROM dbo.outbox_events WHERE event_id = @p1"
			}
			if err := db.QueryRowContext(context.Background(), query, kafkaPublisher.events[0].EventID).Scan(&status); err != nil {
				t.Fatal(err)
			}
			if status != "SENT" {
				t.Fatalf("expected SENT, got %s", status)
			}
		})
	}
}
