package headers

/*
Cache-Control keys and values https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#cache_directives
*/

const (
	Auth         string = "Authorization"
	Age          string = "Age"
	CacheControl string = "Cache-Control"
)

const (
	MaxAge       string = "max-age"
	MaxStale     string = "max-stale"
	MinFresh     string = "min-fresh"
	NoCache      string = "no-cache"
	NoStore      string = "no-store"
	NoTransform  string = "no-transform"
	OnlyIfCached string = "only-if-cached"
)
