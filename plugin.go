package cache

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/roadrunner-server/api/v2/plugins/cache"
	"github.com/roadrunner-server/api/v2/plugins/config"
	"github.com/roadrunner-server/cache/v2/directives"
	"github.com/roadrunner-server/cache/v2/headers"
	"github.com/roadrunner-server/cache/v2/requests"
	"github.com/roadrunner-server/cache/v2/storage"
	endure "github.com/roadrunner-server/endure/pkg/container"
	"github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/sdk/v2/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	jprop "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	root string = "http"
	name string = "cache"
)

type Plugin struct {
	rh          *requests.Requests
	propagators propagation.TextMapPropagator

	log   *zap.Logger
	cfg   *Config
	cache cache.Cache
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

	p.propagators = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}, jprop.Jaeger{})

	return nil
}

func (p *Plugin) Serve() chan error {
	errCh := make(chan error, 1)

	if p.cache == nil {
		errCh <- errors.Str("no cache backends registered")
		return errCh
	}

	p.rh = requests.NewRequestsHandler(p.cache, storage.NewStorage(), p.log)

	return errCh
}

func (p *Plugin) Stop() error {
	return nil
}

func (p *Plugin) Collects() []interface{} {
	return []interface{}{
		p.GetCacheBackend,
	}
}

func (p *Plugin) GetCacheBackend(_ endure.Named, cache cache.HTTPCacheFromConfig) {
	var err error
	p.cache, err = cache.FromConfig(p.log)
	if err != nil {
		p.log.Error("cache construct", zap.Error(err))
	}
}

func (p *Plugin) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if val, ok := r.Context().Value(utils.OtelTracerNameKey).(string); ok {
			tp := trace.SpanFromContext(r.Context()).TracerProvider()
			ctx, span := tp.Tracer(val, trace.WithSchemaURL(semconv.SchemaURL),
				trace.WithInstrumentationVersion(otelhttp.SemVersion())).
				Start(r.Context(), name, trace.WithSpanKind(trace.SpanKindServer))
			defer span.End()

			// inject
			p.propagators.Inject(ctx, propagation.HeaderCarrier(r.Header))
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

		start := time.Now()

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
			p.rh.GET(w, r, next, cc, start)
			return
		case http.MethodHead:
			next.ServeHTTP(w, r)
		case http.MethodPost:
			next.ServeHTTP(w, r)
		default:
			// passthrough request to the worker for other methods
			next.ServeHTTP(w, r)
		}
	})
}

func (p *Plugin) Name() string {
	return name
}
