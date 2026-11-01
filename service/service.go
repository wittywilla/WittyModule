// Package service is how a module that stays running reaches a server: a process of its own
// that listens on this machine only, with a secret for each start, where calls can overlap and
// a browser's connection can be passed straight through. It is for what a pipe with one call at a
// time can't do, such as a game: everything else is a helper. Both ends are here, and nothing
// else, so a module and a server share this package and no code of each other's.
package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/wittywilla/wittymodule"
)

// TokenHeader carries the start's secret on every request.
const TokenHeader = "X-Witty-Service-Token"

// MethodStart is the helper call that says where the service is.
const MethodStart = "service"

// CallBusy asks a service whether it has work going on, before it's stopped for being idle.
const CallBusy = "service.busy"

// Started is the answer to it.
type Started struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// Handler answers one call from a server: with its results, and with the receiver if the call
// changed it.
type Handler func(ctx context.Context, recv json.RawMessage, args []json.RawMessage) (rets []any, back any, err error)

// Config is a service module's side of things.
type Config struct {
	Manifest []byte
	Handlers map[string]Handler
	// ErrorCode names an error the server has its own copy of.
	ErrorCode func(error) string
	// Routes adds whatever isn't a call, such as a stream a browser connects to.
	Routes func(mux *http.ServeMux)
	// OnStop runs when the server goes away, before the process does.
	OnStop func()
	// Busy says it has work going on, such as a game being played, with nobody calling.
	Busy func() bool
}

// Main runs a service module. Started by hand, it explains itself instead of waiting silently.
func Main(c Config) {
	m, err := wittymodule.ParseManifest(c.Manifest)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(os.Args) != 2 || os.Args[1] != wittymodule.HelperFlag {
		fmt.Fprintf(os.Stderr, "%s %s is a service that WittyMediaServer starts by itself; there is nothing to do by hand.\n", m.Name, m.Version)
		os.Exit(2)
	}
	started, err := listen(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Leaving when the server closes the pipe means it can't outlive the server.
	err = wittymodule.Serve(os.Stdin, os.Stdout, wittymodule.Hello{Module: m.ID, Version: m.Version, API: m.API}, func(method string, _ json.RawMessage) (any, []byte, error) {
		if method == MethodStart {
			return started, nil, nil
		}
		return nil, nil, &wittymodule.CallError{Code: "unknown-method", Message: fmt.Sprintf("this version has no %q", method)}
	})
	if c.OnStop != nil {
		c.OnStop()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func listen(c Config) (*Started, error) {
	if c.Busy != nil {
		handlers := make(map[string]Handler, len(c.Handlers)+1)
		for k, v := range c.Handlers {
			handlers[k] = v
		}
		handlers[CallBusy] = func(context.Context, json.RawMessage, []json.RawMessage) ([]any, any, error) {
			return []any{c.Busy()}, nil, nil
		}
		c.Handlers = handlers
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(secret)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("POST /call/{name}", callHandler(c))
	if c.Routes != nil {
		c.Routes(mux)
	}
	srv := &http.Server{Handler: guard(token, mux), ReadHeaderTimeout: 30 * time.Second}
	go func() { _ = srv.Serve(l) }()
	return &Started{URL: "http://" + l.Addr().String(), Token: token}, nil
}

// guard lets in only whoever was told this start's secret, which is the server: a port on this
// machine can be reached by anything else running on it.
func guard(token string, next http.Handler) http.Handler {
	want := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get(TokenHeader)), want) != 1 {
			http.Error(w, "no", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// maxCall limits what a call can bring: a server sends settings and names, never media.
const maxCall = 64 << 20

type callBody struct {
	Recv json.RawMessage   `json:"recv,omitempty"`
	Args []json.RawMessage `json:"args,omitempty"`
}

type callAnswer struct {
	Rets []json.RawMessage `json:"rets,omitempty"`
	Back json.RawMessage   `json:"back,omitempty"`
	Err  string            `json:"err,omitempty"`
	Code string            `json:"code,omitempty"`
}

func callHandler(c Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := c.Handlers[r.PathValue("name")]
		if h == nil {
			writeAnswer(w, callAnswer{Err: "this version has no " + r.PathValue("name"), Code: "unknown-method"})
			return
		}
		var b callBody
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCall)).Decode(&b); err != nil {
			writeAnswer(w, callAnswer{Err: "the call is unreadable", Code: "bad-call"})
			return
		}
		rets, back, err := safely(h, r.Context(), b)
		var a callAnswer
		if err != nil {
			a.Err = err.Error()
			if c.ErrorCode != nil {
				a.Code = c.ErrorCode(err)
			}
		}
		for _, v := range rets {
			raw, merr := Marshal(v)
			if merr != nil {
				a = callAnswer{Err: "the answer can't be sent: " + merr.Error()}
				break
			}
			a.Rets = append(a.Rets, raw)
		}
		if back != nil {
			if raw, merr := Marshal(back); merr == nil {
				a.Back = raw
			}
		}
		writeAnswer(w, a)
	})
}

// safely keeps one bad call from ending a service that has games running.
func safely(h Handler, ctx context.Context, b callBody) (rets []any, back any, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("the call failed: %v", p)
		}
	}()
	return h(ctx, b.Recv, b.Args)
}

func writeAnswer(w http.ResponseWriter, a callAnswer) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(a)
}

// Arg reads one argument of a call. One the server left off stays at its zero value.
func Arg(args []json.RawMessage, i int, v any) error {
	if i >= len(args) || len(args[i]) == 0 {
		return nil
	}
	if err := Unmarshal(args[i], v); err != nil {
		return fmt.Errorf("argument %d is unreadable: %w", i, err)
	}
	return nil
}

// Recv reads a call's receiver.
func Recv(raw json.RawMessage, v any) error {
	if len(raw) == 0 {
		return nil
	}
	if err := Unmarshal(raw, v); err != nil {
		return fmt.Errorf("the call's receiver is unreadable: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------- the server's end

// Conn reaches a running service.
type Conn struct {
	Base   *url.URL
	Token  string
	Client *http.Client
	// Limit is what a call may take with no deadline of its own; 0 means the default.
	Limit time.Duration
}

// DefaultLimit is what a call may take: a game loading can take a while, and a call with a
// deadline of its own has that instead.
const DefaultLimit = 3 * time.Minute

// Dial finds the service of a module, starting it if need be. A server sets it as it starts,
// and without it, or without the module, calls fail with what it returned.
var Dial func(module string) (*Conn, error)

// What else a server sets as it starts.
var (
	// Running says a module's service is running now, without starting it.
	Running func(module string) bool
	// Proxy passes a request straight through to a path of a module's service.
	Proxy func(module string, w http.ResponseWriter, r *http.Request, path string) error
	// Subscribe hears one kind of thing a module's service says unasked, until it next starts.
	Subscribe func(module, method string, fn func(json.RawMessage)) error
)

var (
	startMu sync.Mutex
	onStart = map[string][]func(*Conn){}
)

// OnStart runs something each time a module's service starts: a service that stopped has
// forgotten anything it was told.
func OnStart(module string, fn func(*Conn)) {
	startMu.Lock()
	defer startMu.Unlock()
	onStart[module] = append(onStart[module], fn)
}

// RunStarted runs what OnStart was given. A server calls it with a new connection.
func RunStarted(module string, c *Conn) {
	startMu.Lock()
	fns := slices.Clone(onStart[module])
	startMu.Unlock()
	for _, fn := range fns {
		fn(c)
	}
}

var (
	errMu   sync.Mutex
	errCode = map[string]error{}
)

// RegisterError says which error a code means here.
func RegisterError(code string, err error) {
	errMu.Lock()
	defer errMu.Unlock()
	errCode[code] = err
}

// CallError is a service saying no, with its words.
type CallError struct {
	Code    string
	Message string
	known   error
}

func (e *CallError) Error() string { return e.Message }
func (e *CallError) Unwrap() error { return e.known }

// ErrNoService: nothing has been set up to reach a service with.
var ErrNoService = errors.New("this module isn't on this server")

// Call makes a call to a module's service. back is the receiver to update, and out are where the
// results go. It returns the module's error, or why it couldn't be asked.
func Call(ctx context.Context, module, name string, recv, back any, args []any, out []any) error {
	if Dial == nil {
		return ErrNoService
	}
	c, err := Dial(module)
	if err != nil {
		return err
	}
	return c.Call(ctx, name, recv, back, args, out)
}

// Call makes a call on this connection.
func (c *Conn) Call(ctx context.Context, name string, recv, back any, args []any, out []any) error {
	body := callBody{}
	var err error
	if recv != nil {
		if body.Recv, err = Marshal(recv); err != nil {
			return err
		}
	}
	for _, a := range args {
		raw, err := Marshal(a)
		if err != nil {
			return err
		}
		body.Args = append(body.Args, raw)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if _, ok := ctx.Deadline(); !ok {
		limit := c.Limit
		if limit == 0 {
			limit = DefaultLimit
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, limit)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.Base.JoinPath("call", name).String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set(TokenHeader, c.Token)
	res, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("the module's service answered %s", res.Status)
	}
	var a callAnswer
	if err := json.NewDecoder(io.LimitReader(res.Body, wittymodule.MaxBlob)).Decode(&a); err != nil {
		return fmt.Errorf("the module's answer is unreadable: %w", err)
	}
	for i, v := range out {
		if i < len(a.Rets) {
			if err := Unmarshal(a.Rets[i], v); err != nil {
				return fmt.Errorf("the module's answer is unreadable: %w", err)
			}
		}
	}
	if back != nil && len(a.Back) > 0 {
		if err := Unmarshal(a.Back, back); err != nil {
			return fmt.Errorf("the module's answer is unreadable: %w", err)
		}
	}
	if a.Err != "" {
		errMu.Lock()
		known := errCode[a.Code]
		errMu.Unlock()
		return &CallError{Code: a.Code, Message: a.Err, known: known}
	}
	return nil
}
