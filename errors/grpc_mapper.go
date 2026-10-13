package errors

import (
	"context"
	stderrors "errors"
	"strconv"
	"strings"

	errdetails "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

const errorInfoDomain = "go-core"

type GRPCErrorInfo struct {
	Code      string
	Class     Class
	Finality  Finality
	Retryable bool
}

func ToGRPC(err error) error {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if !stderrors.As(err, &appErr) {
		def, _ := Resolve("CORE-9000")
		return grpcError(def, messageFor(context.Background(), def.Code, def), nil)
	}

	def, registered := Resolve(appErr.Code)
	if !registered {
		appErr = &AppError{Code: def.Code, Message: def.DefaultMessage, Details: appErr.Details}
	}
	message := messageFor(context.Background(), def.Code, def)
	return grpcError(def, message, appErr.Details)
}

func grpcError(def Def, message string, details []Detail) error {
	st := status.New(def.GRPCCode, message)
	stWithInfo, detailErr := st.WithDetails(&errdetails.ErrorInfo{
		Reason: def.Code,
		Domain: errorInfoDomain,
		Metadata: map[string]string{
			"class":     string(def.Class),
			"finality":  string(def.Finality),
			"retryable": strconv.FormatBool(def.Retryable),
		},
	})
	if detailErr == nil {
		st = stWithInfo
	}

	details = normalizeDetails(details)
	if def.Code == "CORE-1000" && len(details) > 0 {
		br := &errdetails.BadRequest{
			FieldViolations: make([]*errdetails.BadRequest_FieldViolation, 0, len(details)),
		}
		for _, d := range details {
			br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       d.Field,
				Description: d.Reason,
			})
		}
		stWithValidation, err := st.WithDetails(br)
		if err == nil {
			st = stWithValidation
		}
	}

	return st.Err()
}

func InfoFromGRPC(err error) (GRPCErrorInfo, bool) {
	st, ok := status.FromError(err)
	if !ok {
		return GRPCErrorInfo{}, false
	}

	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok && info != nil {
			if info.Domain != errorInfoDomain {
				continue
			}
			def, registered := Lookup(info.Reason)
			if !registered {
				return GRPCErrorInfo{}, false
			}
			retryable, _ := strconv.ParseBool(info.Metadata["retryable"])
			return GRPCErrorInfo{Code: def.Code, Class: Class(info.Metadata["class"]), Finality: Finality(info.Metadata["finality"]), Retryable: retryable}, true
		}
	}

	return GRPCErrorInfo{}, false
}

func RetryableFromGRPC(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok && info != nil {
			if retryableStr, ok := info.Metadata["retryable"]; ok {
				retryable, _ := strconv.ParseBool(retryableStr)
				return retryable
			}
		}
	}
	return false
}

func DetailsFromGRPC(err error) []Detail {
	st, ok := status.FromError(err)
	if !ok {
		return nil
	}

	var out []Detail
	for _, d := range st.Details() {
		br, ok := d.(*errdetails.BadRequest)
		if !ok || br == nil {
			continue
		}
		for _, fv := range br.FieldViolations {
			if fv == nil {
				continue
			}
			out = append(out, Detail{
				Field:  fv.Field,
				Reason: fv.Description,
			})
		}
	}
	return out
}

func normalizeDetails(in []Detail) []Detail {
	if len(in) == 0 {
		return nil
	}

	out := make([]Detail, 0, len(in))
	for _, d := range in {
		field := strings.TrimSpace(d.Field)
		reason := strings.TrimSpace(d.Reason)
		if field == "" && reason == "" {
			continue
		}
		out = append(out, Detail{
			Field:  field,
			Reason: reason,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
