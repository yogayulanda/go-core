package logger

import "context"

// EventLog represents important domain or system events.
//
// Used for tracking significant actions such as:
// - user_login
// - account_blocked
// - role_changed
// - approval_action
//
// This log is intended for event tracking & compliance visibility.
//
// ⚠ DO NOT log sensitive data such as:
// - password
// - token
// - full request body
//
// Example:
//
//	EventLog{
//	    EventType: "user_login",
//	    ActorID:   "user_12345",
//	    Resource:  "mobile_app",
//	    Status:    "success",
//	    Metadata: map[string]interface{}{
//	        "ip_address": "103.25.10.20",
//	        "device":     "ios",
//	    },
//	}
type EventLog struct {
	Category string
	Action   string
	Outcome  string
	Metadata map[string]interface{}
}

func (z *zapLogger) LogEvent(ctx context.Context, e EventLog) {
	fields := []Field{
		{Key: "event.kind", Value: "event"},
		{Key: "event.category", Value: e.Category},
		{Key: "event.action", Value: e.Action},
		{Key: "event.outcome", Value: e.Outcome},
		{Key: "app.log_type", Value: "event"},
	}

	if e.Metadata != nil {
		fields = append(fields, Field{Key: "app.metadata", Value: e.Metadata})
	}

	z.Info(ctx, "event "+e.Action+" "+e.Outcome, fields...)
}
