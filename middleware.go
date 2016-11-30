package jsonserv

import (
	"compress/gzip"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	MaxBodySize               = "max_body_size"
	StartTime                 = "start_time"
	DebugFlag                 = "debug"
	headerContentEncoding     = "Content-Encoding"
	headerContentEncodingGzip = "gzip"
	headerAcceptEncoding      = "Accept-Encoding"
	headerAcceptEncodingGzip  = "gzip"
)

// instance of middleware

type Middleware interface {
	Ingress(ctx interface{}, req *Request, res *Response)
	Egress(ctx interface{}, req *Request, res *Response)
}

// collection of middleware

type middlewares []Middleware

func (m middlewares) Ingress(ctx interface{}, req *Request, res *Response) {
	for _, middleware := range m {
		middleware.Ingress(ctx, req, res)
	}
}

func (m middlewares) Egress(ctx interface{}, req *Request, res *Response) {
	for i := len(m) - 1; i >= 0; i-- {
		m[i].Egress(ctx, req, res)
	}
}

// static value middleware

type staticValueMiddleware struct {
	key   string
	value interface{}
}

func NewStaticValueMiddleware(key string, value interface{}) Middleware {
	return &staticValueMiddleware{
		key:   key,
		value: value,
	}
}

func (m staticValueMiddleware) Ingress(ctx interface{}, req *Request, res *Response) {
	req.SetMiddlewareVar(m.key, m.value)
}

func (m staticValueMiddleware) Egress(ctx interface{}, req *Request, res *Response) {
}

// debug flag middleware

// NewDebugFlagMiddleware creates a middleware that sets a debug flag
// Debug mode will enable error messages in 500 responses
func NewDebugFlagMiddleware(debug bool) Middleware {
	return NewStaticValueMiddleware(DebugFlag, debug)
}

// max request size middleware

// NewMaxRequestSizeMiddleware creates a middleware that sets the maximum size read of incoming reqs
func NewMaxRequestSizeMiddleware(maxRequestSize int64) Middleware {
	return NewStaticValueMiddleware(MaxBodySize, maxRequestSize)
}

// logging middleware

type loggingMiddleware struct {
	logIngress bool
}

// NewLoggingMiddleware creates middleware that will (optionally) log incoming requests (method, url)
// and will log responses (method, code, url, duration)
func NewLoggingMiddleware(logIngress bool) Middleware {
	return &loggingMiddleware{
		logIngress: logIngress,
	}
}

func (m loggingMiddleware) Ingress(ctx interface{}, req *Request, res *Response) {
	req.SetMiddlewareVar(StartTime, time.Now())
	if m.logIngress {
		log.Printf("← %s %s", req.Method(), req.URL())
	}
}

func (m loggingMiddleware) Egress(ctx interface{}, req *Request, res *Response) {
	start := req.GetMiddlewareVar(StartTime).(time.Time)
	if res.Err != nil {
		log.Printf("→ ERROR %s %d %s (%s): %v", req.Method(), res.Code, req.URL(), time.Now().Sub(start), res.Err)
	} else {
		log.Printf("→ %s %d %s (%s)", req.Method(), res.Code, req.URL(), time.Now().Sub(start))
	}
}

// gzip middleware

type gzipMiddleware struct {
}

// NewGzipMiddleware creates a middleware that will compress responses using
// gzip if the requesting entity supports it via the Accept-Encoding header
func NewGzipMiddleware() Middleware {
	return &gzipMiddleware{}
}

func (m gzipMiddleware) Ingress(ctx interface{}, req *Request, res *Response) {
	if strings.Contains(req.Header().Get(headerAcceptEncoding), headerAcceptEncodingGzip) {
		res.Writer = &gzipWriter{
			response: res.Writer,
			gz:       gzip.NewWriter(res.Writer),
		}
		res.AddHeader(headerContentEncoding, headerContentEncodingGzip)
	}
}

func (m gzipMiddleware) Egress(ctx interface{}, req *Request, res *Response) {
}

type gzipWriter struct {
	response ResponseWriter
	gz       *gzip.Writer
}

func (gz *gzipWriter) Header() http.Header {
	return gz.response.Header()
}

func (gz *gzipWriter) Write(data []byte) (int, error) {
	return gz.gz.Write(data)
}

func (gz *gzipWriter) WriteHeader(code int) {
	gz.response.WriteHeader(code)
}

func (gz *gzipWriter) Close() {
	gz.gz.Close()
}
