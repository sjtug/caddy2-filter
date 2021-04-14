package filter

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"

	"go.uber.org/zap"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(Middleware{})
	httpcaddyfile.RegisterHandlerDirective("filter", parseCaddyfile)
}

var paramReplacementPattern = regexp.MustCompile("\\{[a-zA-Z0-9_\\-.]+?}")

// Middleware implements an HTTP handler that replaces response contents based on regex
//
// Additional configuration is required in addition to adding a filter{} block. See
// Github page for instructions.
type Middleware struct {
	// Regex to specify which kind of response should we filter
	ContentType string `json:"content_type"`
	// Regex to specify which pattern to look up
	SearchPattern string `json:"search_pattern"`
	// A byte-array specifying the string used to replace matches
	Replacement []byte `json:"replacement"`

	MaxSize int    `json:"max_size"`
	Path    string `json:"path"`

	compiledContentTypeRegex *regexp.Regexp
	compiledSearchRegex      *regexp.Regexp
	compiledPathRegex        *regexp.Regexp

	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (Middleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.filter",
		New: func() caddy.Module { return new(Middleware) },
	}
}

const DefaultMaxSize = 2 * 1024 * 1024

// Provision implements caddy.Provisioner.
func (m *Middleware) Provision(ctx caddy.Context) error {
	var err error
	m.logger = ctx.Logger(m)
	m.logger.Debug(fmt.Sprintf("ContentType: %s. SearchPattern: %s",
		m.ContentType,
		m.SearchPattern))
	if m.MaxSize == 0 {
		m.MaxSize = DefaultMaxSize
	}
	if m.Path == "" {
		m.Path = ".*"
	}
	if m.compiledContentTypeRegex, err = regexp.Compile(m.ContentType); err != nil {
		return fmt.Errorf("invalid content_type: %w", err)
	}
	if m.compiledSearchRegex, err = regexp.Compile(m.SearchPattern); err != nil {
		return fmt.Errorf("invalid search_pattern: %w", err)
	}
	if m.compiledPathRegex, err = regexp.Compile(m.Path); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	return nil
}

// Validate implements caddy.Validator.
func (m *Middleware) Validate() error {
	return nil
}

// CappedSizeRecorder is like httptest.ResponseRecorder,
// but with a cap.
//
// When the size of body exceeds cap,
// CappedSizeRecorder flushes all contents in ResponseRecorder
// together with all subsequent writes into the ResponseWriter
type CappedSizeRecorder struct {
	overflowed bool
	recorder   *httptest.ResponseRecorder
	w          http.ResponseWriter
	cap        int
}

func NewCappedSizeRecorder(cap int, w http.ResponseWriter) *CappedSizeRecorder {
	return &CappedSizeRecorder{
		overflowed: false,
		recorder:   httptest.NewRecorder(),
		w:          w,
		cap:        cap,
	}
}

func (csr *CappedSizeRecorder) Overflowed() bool {
	return csr.overflowed
}

func (csr *CappedSizeRecorder) Header() http.Header {
	return csr.recorder.Header()
}

func (csr *CappedSizeRecorder) FlushHeaders() {
	for k, vs := range csr.recorder.Header() {
		for _, v := range vs {
			csr.w.Header().Add(k, v)
		}
	}
	csr.w.WriteHeader(csr.recorder.Code)
}

// Flush contents to writer
func (csr *CappedSizeRecorder) Flush() (int64, error) {
	if !csr.overflowed {
		log.Fatal("Flush called when overflowed is false")
	}
	csr.FlushHeaders()
	return io.Copy(csr.w, csr.recorder.Body)
}

func (csr *CappedSizeRecorder) Recorder() *httptest.ResponseRecorder {
	if csr.overflowed {
		log.Fatal("trying to get Recorder when overflowed")
	}
	return csr.recorder
}

func (csr *CappedSizeRecorder) Write(b []byte) (int, error) {
	if !csr.overflowed && len(b)+csr.recorder.Body.Len() > csr.cap {
		csr.overflowed = true
		if written, err := csr.Flush(); err != nil {
			return int(written), err
		}
	}
	if csr.overflowed {
		return csr.w.Write(b)
	} else {
		return csr.recorder.Write(b)
	}
}

func (csr *CappedSizeRecorder) WriteHeader(statusCode int) {
	if csr.overflowed {
		log.Fatal("CappedSizeRecorder overflowed on WriteHeader")
	}
	csr.recorder.WriteHeader(statusCode)
}

// Performs the replacement of each placeholder in the previously matched response fragment
// (matched using SearchRegex).
func (m Middleware) replacer(input []byte, repl *caddy.Replacer) []byte {
	pattern := m.compiledSearchRegex
	if pattern == nil {
		return input
	}

	if len(m.Replacement) <= 0 {
		return []byte{}
	}

	groups := pattern.FindSubmatch(input)
	replacement := paramReplacementPattern.ReplaceAllFunc(m.Replacement, func(input2 []byte) []byte {
		return m.paramReplacer(input2, groups, repl) // forward the placeholder replacement
	})
	return replacement
}

// This method supports two types of placeholders:
//  - {N} where N matches any capturing group of SearchRegex
//  - {X} where X is any key available in caddy.Replacer
func (m Middleware) paramReplacer(input []byte, groups [][]byte, repl *caddy.Replacer) []byte {
	if len(input) < 3 {
		return input
	}

	// replace based on a capturing group
	name := string(input[1 : len(input)-1])
	if index, err := strconv.Atoi(name); err == nil {
		if index >= 0 && index < len(groups) {
			return groups[index]
		}
		return input
	}

	// replace based on caddy's replacer
	if value, found := repl.GetString(name); found {
		return []byte(value)
	}

	return input
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if !m.compiledPathRegex.MatchString(r.URL.Path) {
		return next.ServeHTTP(w, r)
	}
	csr := NewCappedSizeRecorder(m.MaxSize, w)
	nextErr := next.ServeHTTP(csr, r)
	if csr.Overflowed() {
		return nextErr
	}
	csr.FlushHeaders()
	if m.compiledContentTypeRegex.MatchString(csr.Recorder().Result().Header.Get("Content-Type")) {
		buf := new(bytes.Buffer)
		repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
		if _, err := buf.ReadFrom(csr.Recorder().Result().Body); err != nil {
			return fmt.Errorf("failed to read from response body: %w", err)
		}
		replaced := m.compiledSearchRegex.ReplaceAllFunc(buf.Bytes(), func(input []byte) []byte {
			return m.replacer(input, repl) // forward the replacement processing
		})
		if _, err := io.Copy(w, bytes.NewReader(replaced)); err != nil {
			return fmt.Errorf("error when copying replaced response body: %w", err)
		}
	} else {
		if _, err := io.Copy(w, csr.recorder.Body); err != nil {
			return fmt.Errorf("error when copying response body: %w", err)
		}
	}
	return nextErr
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (m *Middleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	if !d.Next() {
		return d.Err("expected token following filter")
	}
	for d.NextBlock(0) {
		key := d.Val()
		var value string
		d.Args(&value)
		if d.NextArg() {
			return d.ArgErr()
		}
		switch key {
		case "content_type":
			m.ContentType = value
		case "search_pattern":
			m.SearchPattern = value
		case "replacement":
			m.Replacement = []byte(value)
		case "max_size":
			val, err := strconv.Atoi(value)
			if err != nil {
				_ = d.Err(fmt.Sprintf("max_size error: %s", err.Error()))
			}
			m.MaxSize = val
		case "path":
			m.Path = value
		default:
			return d.Err(fmt.Sprintf("invalid key for filter directive: %s", key))
		}
	}
	return nil
}

// parseCaddyfile unmarshals tokens from h into a new Middleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m Middleware
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return m, err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*Middleware)(nil)
	_ caddy.Validator             = (*Middleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*Middleware)(nil)
	_ caddyfile.Unmarshaler       = (*Middleware)(nil)
)
