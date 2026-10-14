package response

import (
	"context"
	"net/http"
	"strings"

	coreerrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const codeMetadataKey = "x-response-code"

func SetCode(ctx context.Context, code string) error {
	def, _ := coreerrors.Resolve(code)
	original := code
	if registeredDef, registered := coreerrors.Lookup(code); registered {
		def = registeredDef
		original = ""
	}
	if err := observability.SetResponseInfo(ctx, def.Code, string(def.Class), original); err != nil {
		return err
	}
	return grpc.SetHeader(ctx, metadata.Pairs(codeMetadataKey, def.Code))
}

func CodeFromHeader(md metadata.MD) string {
	values := md.Get(codeMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func CodeFromHTTPHeader(header http.Header) string {
	for key, values := range header {
		if strings.EqualFold(key, "X-Response-Code") || strings.EqualFold(key, "Grpc-Metadata-X-Response-Code") {
			if len(values) > 0 {
				return values[0]
			}
		}
	}
	return ""
}
