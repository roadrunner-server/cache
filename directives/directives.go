package directives

import (
	"strconv"
	"strings"

	"github.com/roadrunner-server/cache/v2/headers"
	"github.com/roadrunner-server/cache/v2/requests"
	"github.com/roadrunner-server/sdk/v2/utils"
	"go.uber.org/zap"
)

/*
Response Cache-Control Directives: https://datatracker.ietf.org/doc/html/rfc7234#section-5.2.2
*/

/*
   Cache-Control   = 1#cache-directive
   cache-directive = token [ "=" ( token / quoted-string ) ]
*/

const (
	eq    byte   = '='
	space string = " "
)

func ParseRequestCacheControl(directives string, log *zap.Logger, r *requests.CacheControl) { //nolint:gocyclo
	split := strings.Split(directives, ",")

	// a lot of allocations here - 4 todo(rustatian): FIXME
	for i := 0; i < len(split); i++ {
		// max-age, max-stale, min-fresh
		if idx := strings.IndexByte(split[i], eq); idx != -1 {
			// get token and associated value
			if len(split[i]) < idx+1 {
				log.Warn("bad header", zap.String("value", split[i]))
				continue
			}

			token := strings.Trim(split[i][:idx], space)
			val := strings.Trim(split[i][idx+1:], space)
			if val == "" || token == "" {
				log.Warn("bad header", zap.String("value", split[i]))
				continue
			}

			switch token {
			case headers.MaxAge:
				valUint, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					log.Error("parse max-age", zap.String("value", val), zap.Error(err))
					continue
				}
				r.MaxAge = utils.Uint64(valUint)
			case headers.MaxStale:
				valUint, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					log.Error("parse max-stale", zap.String("value", val), zap.Error(err))
					continue
				}
				r.MaxStale = utils.Uint64(valUint)
			case headers.MinFresh:
				valUint, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					log.Error("parse min-fresh", zap.String("value", val), zap.Error(err))
					continue
				}
				r.MinFresh = utils.Uint64(valUint)
			}

			continue
		}

		// single tokens
		token := strings.Trim(split[i], space)

		switch token {
		case headers.NoCache:
			r.NoCache = true
		case headers.NoStore:
			r.NoStore = true
		case headers.NoTransform:
			r.NoTransform = true
		case headers.OnlyIfCached:
			r.OnlyIfCached = true
		default:
			continue
		}
	}
}
