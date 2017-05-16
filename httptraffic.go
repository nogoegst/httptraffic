// httptraffic.go - count traffic sent via HTTP handler.
//
// To the extent possible under law, Ivan Markin waived all copyright
// and related or neighboring rights to httptraffic, using the creative
// commons "cc0" public domain dedication. See LICENSE or
// <http://creativecommons.org/publicdomain/zero/1.0/> for full details.

package httptraffic

import (
	"log"
	"net/http"
)

type ResponseWriter struct {
	w               http.ResponseWriter
	trafficCallback func(int)
}

func NewResponseWriter(w http.ResponseWriter, trafficCallback func(int)) *ResponseWriter {
	return &ResponseWriter{
		w:               w,
		trafficCallback: trafficCallback,
	}
}

func (tw *ResponseWriter) Header() http.Header {
	return tw.w.Header()
}

func (tw *ResponseWriter) Write(b []byte) (int, error) {
	n, err := tw.w.Write(b)
	tw.trafficCallback(n)
	return n, err
}

func (tw *ResponseWriter) WriteHeader(h int) {
	tw.w.WriteHeader(h)
}

type KeyWritten struct {
	Key          interface{}
	BytesWritten int64
}

type Handler struct {
	h              http.Handler
	keyFromRequest func(*http.Request) (interface{}, error)
	C              chan KeyWritten
}

func NewHandler(h http.Handler, keyFromRequest func(*http.Request) (interface{}, error)) *Handler {
	return &Handler{
		h:              h,
		keyFromRequest: keyFromRequest,
		C:              make(chan KeyWritten),
	}
}

func (th *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key, err := th.keyFromRequest(r)
	if err != nil {
		log.Print(err) // XXX send to some chan
		return
	}
	bytesSent := int64(0)
	wch := make(chan int64)
	go func() {
		for {
			w := <-wch
			if w == -1 {
				break
			}
			bytesSent += w
		}
		th.C <- KeyWritten{Key: key, BytesWritten: bytesSent}
	}()
	defer func() {
		wch <- int64(-1)
	}()

	tw := NewResponseWriter(w, func(b int) {
		wch <- int64(b)

	})
	th.h.ServeHTTP(tw, r)
}
