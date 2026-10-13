package observability

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

const LabelMetadataPrefix = "x-label-"

type fieldBagContextKey struct{}

type FieldBag struct {
	mu                sync.RWMutex
	userID            string
	labels            map[string]string
	requestIDReplaced bool
}

var propagatedLabels = struct {
	sync.RWMutex
	names map[string]struct{}
}{names: make(map[string]struct{})}

func WithFieldBag(ctx context.Context) context.Context {
	if _, ok := ctx.Value(fieldBagContextKey{}).(*FieldBag); ok {
		return ctx
	}
	return context.WithValue(ctx, fieldBagContextKey{}, &FieldBag{labels: make(map[string]string)})
}

func EnsureFieldBag(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return WithFieldBag(ctx)
}

func fieldBag(ctx context.Context) *FieldBag {
	if ctx == nil {
		return nil
	}
	bag, _ := ctx.Value(fieldBagContextKey{}).(*FieldBag)
	return bag
}

func SetUserID(ctx context.Context, userID string) error {
	bag := fieldBag(ctx)
	if bag == nil {
		return fmt.Errorf("observability field bag is not present in context")
	}
	bag.mu.Lock()
	bag.userID = userID
	bag.mu.Unlock()
	return nil
}

func GetUserID(ctx context.Context) string {
	bag := fieldBag(ctx)
	if bag == nil {
		return ""
	}
	bag.mu.RLock()
	defer bag.mu.RUnlock()
	return bag.userID
}

func SetLabel(ctx context.Context, name, value string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " .") {
		return fmt.Errorf("invalid context label name %q", name)
	}
	if name == "request_id" || name == "user_id" {
		return fmt.Errorf("context label %q is reserved", name)
	}
	bag := fieldBag(ctx)
	if bag == nil {
		return fmt.Errorf("observability field bag is not present in context")
	}
	bag.mu.Lock()
	bag.labels[name] = value
	bag.mu.Unlock()
	return nil
}

func Labels(ctx context.Context) map[string]string {
	bag := fieldBag(ctx)
	if bag == nil {
		return nil
	}
	bag.mu.RLock()
	defer bag.mu.RUnlock()
	labels := make(map[string]string, len(bag.labels))
	for key, value := range bag.labels {
		labels[key] = value
	}
	return labels
}

func RegisterPropagatedLabel(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " .") || name == "request_id" || name == "user_id" {
		return fmt.Errorf("invalid propagated label name %q", name)
	}
	propagatedLabels.Lock()
	propagatedLabels.names[name] = struct{}{}
	propagatedLabels.Unlock()
	return nil
}

func PropagatedLabels() []string {
	propagatedLabels.RLock()
	defer propagatedLabels.RUnlock()
	names := make([]string, 0, len(propagatedLabels.names))
	for name := range propagatedLabels.names {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func RestoreIncomingIdentity(ctx context.Context, userID string, labels map[string]string) {
	if userID != "" {
		_ = SetUserID(ctx, userID)
	}
	for name, value := range labels {
		_ = SetLabel(ctx, name, value)
	}
}

func MarkRequestIDReplaced(ctx context.Context) {
	if bag := fieldBag(ctx); bag != nil {
		bag.mu.Lock()
		bag.requestIDReplaced = true
		bag.mu.Unlock()
	}
}

func RequestIDReplaced(ctx context.Context) bool {
	bag := fieldBag(ctx)
	if bag == nil {
		return false
	}
	bag.mu.RLock()
	defer bag.mu.RUnlock()
	return bag.requestIDReplaced
}
