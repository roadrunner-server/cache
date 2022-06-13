package storage

import (
	"net/http"
	"sync"
	"time"

	"github.com/roadrunner-server/api/v2/plugins/cache"
	"github.com/roadrunner-server/cache/v2/writer"
	cacheV1beta "go.buf.build/protocolbuffers/go/roadrunner-server/api/proto/cache/v1beta"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Storage struct {
	rspPool sync.Pool
}

func NewStorage() *Storage {
	return &Storage{
		rspPool: sync.Pool{
			New: func() interface{} {
				return new(cacheV1beta.Response)
			},
		},
	}
}

func (s *Storage) Write(wr *writer.Writer, log *zap.Logger, cache cache.Cache, id uint64, start time.Time) {
	/*
		First - check the status code, should be only 200, 203, 204, 206, 300, 301, 404, 405, 410, 414, and 501
	*/
	switch wr.Code {
	case http.StatusOK:
		s.storeGet(wr, log, cache, id, start)
		return
	case http.StatusNonAuthoritativeInfo:
	case http.StatusNoContent:
	case http.StatusPartialContent:
	case http.StatusMultipleChoices:
	case http.StatusMovedPermanently:
	case http.StatusNotFound:
	case http.StatusMethodNotAllowed:
	case http.StatusGone:
	case http.StatusRequestURITooLong:
	case http.StatusNotImplemented:
	}
}

func (s *Storage) storeGet(wr *writer.Writer, log *zap.Logger, cache cache.Cache, id uint64, start time.Time) {
	payload := s.getRsp()
	defer s.putRsp(payload)

	payload.Headers = make(map[string]*cacheV1beta.HeaderValue, len(wr.HdrToSend))
	payload.Code = uint64(wr.Code)
	payload.Data = make([]byte, len(wr.Data))

	// https://datatracker.ietf.org/doc/html/rfc7234#section-4.2.3
	payload.Timestamp = start.Format(time.RFC3339)
	copy(payload.Data, wr.Data)

	for k := range wr.HdrToSend {
		for i := 0; i < len(wr.HdrToSend[k]); i++ {
			payload.Headers[k].Value = wr.HdrToSend[k]
		}
	}

	data, err := proto.Marshal(payload)
	if err != nil {
		log.Error("cache write", zap.Error(err))
		return
	}

	err = cache.Set(id, data)
	if err != nil {
		log.Error("failed to write cache", zap.Error(err))
	}
}

func (s *Storage) getRsp() *cacheV1beta.Response {
	return s.rspPool.Get().(*cacheV1beta.Response)
}

func (s *Storage) putRsp(r *cacheV1beta.Response) {
	r.Reset()
	s.rspPool.Put(r)
}
