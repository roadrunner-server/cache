package writer

import (
	"net/http"
)

type Writer struct {
	Code      int                 `json:"code"`
	Data      []byte              `json:"data"`
	HdrToSend map[string][]string `json:"headers"`
}

func (w *Writer) WriteHeader(code int) {
	w.Code = code
}

func (w *Writer) Write(b []byte) (int, error) {
	w.Data = make([]byte, len(b))
	copy(w.Data, b)
	return len(w.Data), nil
}

func (w *Writer) Header() http.Header {
	return w.HdrToSend
}
