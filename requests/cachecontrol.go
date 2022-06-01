package requests

// CacheControl represents possible Cache-Control request header values
type CacheControl struct {
	MaxAge   *uint64
	MaxStale *uint64
	MinFresh *uint64

	NoCache      bool
	NoStore      bool
	NoTransform  bool
	OnlyIfCached bool
}

func (r *CacheControl) Reset() {
	r.MaxAge = nil
	r.MaxStale = nil
	r.MinFresh = nil

	r.NoCache = false
	r.NoStore = false
	r.NoTransform = false
	r.OnlyIfCached = false
}
