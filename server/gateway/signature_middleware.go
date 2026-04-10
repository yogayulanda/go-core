package gateway

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/yogayulanda/go-core/app"
	coreErrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

func withSignatureValidation(application *app.App, next http.Handler) http.Handler {
	cfg := application.Config()
	sigCfg := cfg.Auth.Signature

	if !sigCfg.Enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID := observability.GetRequestID(ctx)
		log := application.Logger()

		providedSignature := r.Header.Get(sigCfg.HeaderKey)
		providedTimestampRaw := r.Header.Get(sigCfg.TimestampKey)

		sendUnauthorized := func(reason string) {
			log.LogService(ctx, logger.ServiceLog{
				Operation: "signature_validation",
				Status:    "failed",
				ErrorCode: "unauthorized",
				Metadata: map[string]interface{}{
					"reason": reason,
				},
			})

			w.Header().Set("Content-Type", "application/json")
			if requestID != "" {
				w.Header().Set("x-request-id", requestID)
			}
			w.WriteHeader(http.StatusUnauthorized)

			errResp := coreErrors.ErrorResponse{
				Code:      string(coreErrors.CodeUnauthorized),
				Message:   "unauthorized request",
				RequestID: requestID,
			}
			_ = json.NewEncoder(w).Encode(errResp)
		}

		if providedSignature == "" || providedTimestampRaw == "" {
			sendUnauthorized("missing signature or timestamp headers")
			return
		}

		// Validate Timestamp Replay Attack
		tsInt, err := strconv.ParseInt(providedTimestampRaw, 10, 64)
		if err != nil {
			sendUnauthorized("invalid timestamp format")
			return
		}

		// Assume timestamp is standard Unix seconds
		reqTime := time.Unix(tsInt, 0)
		drift := time.Since(reqTime)

		if drift < -sigCfg.MaxTimeDrift || drift > sigCfg.MaxTimeDrift {
			sendUnauthorized(fmt.Sprintf("timestamp drift exceeded limit: %v", drift))
			return
		}

		// Read Body safely without breaking downstream handlers
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				sendUnauthorized("failed to read request body")
				return
			}
			// Restore the io.ReadCloser to its original state so controllers can reuse it
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Standard concat: Method + ":" + Path + ":" + Timestamp + ":" + Body
		payloadString := r.Method + ":" + r.URL.Path + ":" + providedTimestampRaw + ":" + string(bodyBytes)

		// Calculate HMAC
		mac := hmac.New(sha256.New, []byte(sigCfg.MasterKey))
		mac.Write([]byte(payloadString))
		expectedSignature := hex.EncodeToString(mac.Sum(nil))

		// Compare hashes
		if !hmac.Equal([]byte(providedSignature), []byte(expectedSignature)) {
			sendUnauthorized("signature mismatch")
			return
		}

		// Successful validation
		next.ServeHTTP(w, r)
	})
}
