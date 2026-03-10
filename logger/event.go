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
	EventType string                 // e.g. "user_login", "account_blocked"
	ActorID   string                 // e.g. "user_12345"
	Resource  string                 // e.g. "mobile_app", "account_6789"
	Status    string                 // e.g. "success", "failed"
	Metadata  map[string]interface{} // optional structured additional info
}

func (z *zapLogger) LogEvent(ctx context.Context, e EventLog) {
	fields := []Field{
		{Key: "category", Value: "event"},
		{Key: "event_type", Value: e.EventType},
		{Key: "actor_id", Value: e.ActorID},
		{Key: "resource", Value: e.Resource},
		{Key: "status", Value: e.Status},
	}

	if e.Metadata != nil {
		fields = append(fields, Field{Key: "metadata", Value: e.Metadata})
	}

	z.Info(ctx, "event_log", fields...)
}
