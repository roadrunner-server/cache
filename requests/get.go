package requests

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/roadrunner-server/api/v2/plugins/cache"
	"github.com/roadrunner-server/cache/v2/hasher"
	"github.com/roadrunner-server/cache/v2/headers"
	"github.com/roadrunner-server/cache/v2/storage"
	"github.com/roadrunner-server/cache/v2/writer"
	"github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/sdk/v2/utils"
	cacheV1beta "go.buf.build/protocolbuffers/go/roadrunner-server/api/proto/cache/v1beta"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Requests struct {
	cache   cache.Cache
	storage *storage.Storage
	hs      *hasher.Hasher
	log     *zap.Logger

	wrPool  sync.Pool
	rspPool sync.Pool
	ccPool  sync.Pool
}

func NewRequestsHandler(cache cache.Cache, s *storage.Storage, log *zap.Logger) *Requests {
	return &Requests{
		cache:   cache,
		storage: s,
		log:     log,

		rspPool: sync.Pool{New: func() any {
			return new(cacheV1beta.Response)
		}},

		ccPool: sync.Pool{New: func() any {
			return new(CacheControl)
		}},

		wrPool: sync.Pool{
			New: func() any {
				wr := new(writer.Writer)
				wr.Code = -1
				wr.Data = nil
				wr.HdrToSend = make(map[string][]string, 10)
				return wr
			},
		},
	}
}

// GET https://datatracker.ietf.org/doc/html/rfc7234#section-4
func (req *Requests) GET(w http.ResponseWriter, r *http.Request, next http.Handler, cc *CacheControl) {
	h := req.hs.GetHash()
	defer req.hs.PutHash(h)

	wr := req.getWriter()
	defer req.putWriter(wr)

	pld := req.getRsp()
	defer req.putRsp(pld)

	// write the data to the hash function
	_, err := h.Write(utils.AsBytes(r.RequestURI))
	if err != nil {
		http.Error(w, "failed to write the hash", http.StatusInternalServerError)
		return
	}

	// try to get the data from cache
	out, err := req.cache.Get(h.Sum64())
	if err != nil {
		// cache miss, no data
		if errors.Is(errors.EmptyItem, err) {
			// forward the request to the worker
			next.ServeHTTP(wr, r)
			// send original data to the receiver
			writeResponse(w, wr)
			// handle the response (decide to cache or not)
			req.storage.Write(wr, pld, req.log, req.cache, h.Sum64())
			return
		}

		http.Error(w, "get hash", http.StatusInternalServerError)
		return
	}

	msg := req.getRsp()
	defer req.putRsp(msg)

	err = proto.Unmarshal(out, msg)
	if err != nil {
		http.Error(w, "cache data unpack", http.StatusInternalServerError)
		return
	}

	ts := msg.GetTimestamp()
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		http.Error(w, "timestamp parse", http.StatusInternalServerError)
		return
	}

	ageHdr := time.Since(parsed).Seconds()
	if cc.MaxAge != nil {
		// request should not be accepted
		if uint64(ageHdr) > *cc.MaxAge {
			// delete prev data from the cache
			req.cache.Delete(h.Sum64())
			// serve the request
			next.ServeHTTP(wr, r)
			// write response
			writeResponse(w, wr)
			// write cache
			req.storage.Write(wr, pld, req.log, req.cache, h.Sum64())
			return
		}
	}

	// write Age header
	w.Header().Add(headers.Age, fmt.Sprintf("%.0f", ageHdr))

	// send original data
	for k := range msg.Headers {
		for i := 0; i < len(msg.Headers[k].Value); i++ {
			w.Header().Add(k, msg.Headers[k].Value[i])
		}
	}

	w.WriteHeader(int(msg.Code))
	_, _ = w.Write(msg.Data)
}

func writeResponse(w http.ResponseWriter, wr *writer.Writer) {
	for k := range wr.HdrToSend {
		for kk := range wr.HdrToSend[k] {
			w.Header().Add(k, wr.HdrToSend[k][kk])
		}
	}

	// write the original status code
	w.WriteHeader(wr.Code)
	// write the data
	_, _ = w.Write(wr.Data)
}

func (req *Requests) GetCC() *CacheControl {
	return req.ccPool.Get().(*CacheControl)
}

func (req *Requests) PutCC(cc *CacheControl) {
	cc.Reset()
	req.ccPool.Put(cc)
}

func (req *Requests) getWriter() *writer.Writer {
	wr := req.wrPool.Get().(*writer.Writer)
	return wr
}

func (req *Requests) putWriter(w *writer.Writer) {
	w.Code = -1
	w.Data = nil

	for k := range w.HdrToSend {
		delete(w.HdrToSend, k)
	}

	req.wrPool.Put(w)
}

func (req *Requests) getRsp() *cacheV1beta.Response {
	return req.rspPool.Get().(*cacheV1beta.Response)
}

func (req *Requests) putRsp(r *cacheV1beta.Response) {
	r.Reset()
	req.rspPool.Put(r)
}
