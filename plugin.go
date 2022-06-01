package cache

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/roadrunner-server/api/v2/plugins/cache"
	"github.com/roadrunner-server/api/v2/plugins/config"
	"github.com/roadrunner-server/cache/v2/directives"
	"github.com/roadrunner-server/cache/v2/headers"
	"github.com/roadrunner-server/cache/v2/requests"
	"github.com/roadrunner-server/cache/v2/storage"
	endure "github.com/roadrunner-server/endure/pkg/container"
	"github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/sdk/v2/utils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	root string = "http"
	name string = "cache"
)

type Plugin struct {
	rh *requests.Requests

	log             *zap.Logger
	cfg             *Config
	cache           cache.Cache
	collectedCaches map[string]cache.HTTPCacheFromConfig
}

func (p *Plugin) Init(cfg config.Configurer, log *zap.Logger) error {
	const op = errors.Op("cache_middleware_init")

	if !cfg.Has(fmt.Sprintf("%s.%s", root, name)) {
		return errors.E(op, errors.Disabled)
	}

	err := cfg.UnmarshalKey(fmt.Sprintf("%s.%s", root, name), &p.cfg)
	if err != nil {
		return errors.E(op, err)
	}

	// init default config values
	p.cfg.InitDefaults()

	p.log = new(zap.Logger)
	*p.log = *log
	p.collectedCaches = make(map[string]cache.HTTPCacheFromConfig, 1)

	return nil
}

func (p *Plugin) Serve() chan error {
	errCh := make(chan error, 1)

	if _, ok := p.collectedCaches[p.cfg.Driver]; ok {
		p.cache, _ = p.collectedCaches[p.cfg.Driver].FromConfig(p.log)
		return errCh
	}

	p.rh = requests.NewRequestsHandler(nil, storage.NewStorage(), p.log)

	errCh <- errors.E("no cache drivers registered")
	return errCh
}

func (p *Plugin) Stop() error {
	return nil
}

func (p *Plugin) Collects() []interface{} {
	return []interface{}{
		p.CollectCaches,
	}
}

func (p *Plugin) CollectCaches(name endure.Named, cache cache.HTTPCacheFromConfig) {
	p.collectedCaches[name.Name()] = cache
}

func (p *Plugin) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if val, ok := r.Context().Value(utils.OtelTracerNameKey).(string); ok {
			tp := trace.SpanFromContext(r.Context()).TracerProvider()
			ctx, span := tp.Tracer(val).Start(r.Context(), name)
			defer span.End()
			r = r.WithContext(ctx)
		}

		/*
			https://www.cloudflare.com/en-gb/learning/access-management/what-is-mutual-tls/
			we MUST NOT use a cached response to a request with an Authorization header field
		*/
		if w.Header().Get(headers.Auth) != "" {
			next.ServeHTTP(w, r)
			return
		}

		cc := p.rh.GetCC()
		defer p.rh.PutCC(cc)

		// cwe-117
		cch := r.Header.Get(headers.CacheControl)
		cch = strings.ReplaceAll(cch, "\n", "")
		cch = strings.ReplaceAll(cch, "\r", "")

		/*
		   Cache-Control   = 1#cache-directive
		   cache-directive = token [ "=" ( token / quoted-string ) ]
		*/
		directives.ParseRequestCacheControl(cch, p.log, cc)
		// https://datatracker.ietf.org/doc/html/rfc7234#section-5.2.1.5
		/*
			The "no-store" request directive indicates that a cache MUST NOT
			store any part of either this request or any response to it.
		*/
		if cc.NoCache {
			next.ServeHTTP(w, r)
			return
		}

		switch r.Method {
		/*
			cacheable statuses by default: https://www.rfc-editor.org/rfc/rfc7231#section-6.1
			cacheable methods: https://www.rfc-editor.org/rfc/rfc7231#section-4.2.3 (GET, HEAD, POST (Responses to POST requests are only cacheable when they include explicit freshness information))
		*/
		case http.MethodGet:
			p.rh.GET(w, r, next, cc)
			return
		case http.MethodHead:
			// TODO(rustatian): HEAD method is not supported
			fallthrough
		case http.MethodPost:
			// TODO(rustatian): POST method is not supported
			fallthrough
		default:
			// passthrough request to the worker for other methods
			next.ServeHTTP(w, r)
		}
	})
}

func (p *Plugin) Name() string {
	return name
}
