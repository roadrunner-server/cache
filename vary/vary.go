package vary

import (
	"net/http"

	cacheV1beta "go.buf.build/protocolbuffers/go/roadrunner-server/api/proto/cache/v1beta"
)

// HandleVary https://datatracker.ietf.org/doc/html/rfc7234#section-4.1
func HandleVary(resp *cacheV1beta.Response, r *http.Request) bool {
	return true
}
