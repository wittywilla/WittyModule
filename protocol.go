// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittymodule

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// The helper protocol (module API 1): one line of JSON per message over stdin and
// stdout, optionally followed by a binary blob of the announced size. Sizes are
// checked before anything is read, because the two sides don't trust each other.
const (
	MaxLine = 1 << 20
	MaxBlob = 128 << 20
)

// HelperFlag is what the server starts a helper with.
const HelperFlag = "--wms-helper"

// Hello is the first thing a helper says.
type Hello struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	API     int    `json:"api"`
}

type request struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type response struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
	Code   string          `json:"code,omitempty"`
	Blob   int64           `json:"blob,omitempty"`
}

// notification is what a helper says without being asked (module API 3). It has no id,
// and nothing answers it.
type notification struct {
	Notify string          `json:"notify"`
	Params json.RawMessage `json:"params,omitempty"`
}

// CallError is an answer from a working helper that says no, such as a file it
// can't read. It says nothing about the helper's health; any other error from
// Call means the helper itself failed.
type CallError struct {
	Code    string // for the caller to act on; may be empty
	Message string
}

func (e *CallError) Error() string { return e.Message }

// Handler answers one call with a result to send as JSON, an optional blob, and
// an error (a *CallError keeps its code).
type Handler func(method string, params json.RawMessage) (result any, blob []byte, err error)

func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		return nil, errors.New("a message is too long")
	}
	return line, err
}

func writeLine(w *bufio.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(b) >= MaxLine {
		return errors.New("a message is too long")
	}
	_, _ = w.Write(b)
	return w.WriteByte('\n')
}

// notifier is the running Serve's way out, shared with Notify. A helper has one.
var notifier struct {
	sync.Mutex
	w *bufio.Writer
}

// ErrNotServing is Notify when no Serve is running, or in a process answering with a
// stream, where anything else on the pipe would be taken for part of the stream.
var ErrNotServing = errors.New("this process isn't serving calls, so it has nobody to tell")

// Notify tells the server something unasked, from any goroutine, at any time while Serve
// is running. A server treats it as untrusted, and may ignore it or drop it for sending too
// many. It blocks only while the pipe is full, so don't call it in a tight loop.
func Notify(method string, params any) error {
	if method == "" {
		return errors.New("a notification needs a name")
	}
	b, err := json.Marshal(params)
	if err != nil {
		return err
	}
	notifier.Lock()
	defer notifier.Unlock()
	if notifier.w == nil {
		return ErrNotServing
	}
	if err := writeLine(notifier.w, notification{Notify: method, Params: b}); err != nil {
		return err
	}
	return notifier.w.Flush()
}

// Serve is a helper's loop: it says hello and answers calls until in closes.
func Serve(in io.Reader, out io.Writer, hello Hello, h Handler) error {
	r, w := bufio.NewReaderSize(in, MaxLine), bufio.NewWriter(out)
	if err := writeLine(w, hello); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	notifier.Lock()
	notifier.w = w
	notifier.Unlock()
	defer func() {
		notifier.Lock()
		notifier.w = nil
		notifier.Unlock()
	}()
	for {
		line, err := readLine(r)
		if errors.Is(err, io.EOF) {
			return nil // the server closed the pipe or went away
		}
		if err != nil {
			return err
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			return fmt.Errorf("unreadable call: %w", err)
		}
		res := response{ID: req.ID}
		result, blob, err := safeCall(h, req.Method, req.Params)
		if err != nil {
			res.Error = err.Error()
			var ce *CallError
			if errors.As(err, &ce) {
				res.Code = ce.Code
			}
			blob = nil
		} else if result != nil {
			if res.Result, err = json.Marshal(result); err != nil {
				return err
			}
		}
		if int64(len(blob)) > MaxBlob {
			res.Result, res.Error, blob = nil, "the answer is too large", nil
		}
		res.Blob = int64(len(blob))
		if err := writeAnswer(w, res, blob); err != nil {
			return err
		}
	}
}

// writeAnswer keeps an answer and its blob together, whatever Notify is doing.
func writeAnswer(w *bufio.Writer, res response, blob []byte) error {
	notifier.Lock()
	defer notifier.Unlock()
	if err := writeLine(w, res); err != nil {
		return err
	}
	if _, err := w.Write(blob); err != nil {
		return err
	}
	return w.Flush()
}

// safeCall keeps one bad file from ending a helper: a parser that panics on it
// answers that call with an error, like any other file it can't read.
func safeCall(h Handler, method string, params json.RawMessage) (result any, blob []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			result, blob, err = nil, nil, &CallError{Code: "panic", Message: fmt.Sprintf("%s failed: %v", method, p)}
		}
	}()
	return h(method, params)
}

// Main runs a helper program from its manifest. Started by hand, it explains
// itself instead of waiting silently.
func Main(manifest []byte, h Handler) {
	m, err := ParseManifest(manifest)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(os.Args) != 2 || os.Args[1] != HelperFlag {
		fmt.Fprintf(os.Stderr, "%s %s is a helper that WittyMediaServer starts by itself; there is nothing to do by hand.\n", m.Name, m.Version)
		os.Exit(2)
	}
	if err := Serve(os.Stdin, os.Stdout, Hello{Module: m.ID, Version: m.Version, API: m.API}, h); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// StreamHandler answers one call for a stream with the answer to send, and what to
// write after it. The stream ends when write returns or the server stops reading.
type StreamHandler func(method string, params json.RawMessage) (result any, write func(io.Writer) error, err error)

// ServeStream is Serve for a process that answers one call with a stream: nothing
// else can be said on the pipe afterwards, so the process exits when the stream ends.
func ServeStream(in io.Reader, out io.Writer, hello Hello, h StreamHandler) error {
	r, w := bufio.NewReaderSize(in, MaxLine), bufio.NewWriter(out)
	if err := writeLine(w, hello); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	line, err := readLine(r)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return fmt.Errorf("unreadable call: %w", err)
	}
	res := response{ID: req.ID}
	result, write, err := h(req.Method, req.Params)
	if err == nil && write == nil {
		err = errors.New("no stream")
	}
	if err != nil {
		res.Error = err.Error()
		var ce *CallError
		if errors.As(err, &ce) {
			res.Code = ce.Code
		}
		if werr := writeLine(w, res); werr != nil {
			return werr
		}
		return w.Flush()
	}
	if result != nil {
		if res.Result, err = json.Marshal(result); err != nil {
			return err
		}
	}
	if err := writeLine(w, res); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	// Closing stdin is how the server says stop, so the stream ends with it.
	stop := make(chan error, 1)
	go func() { stop <- write(out) }()
	eof := make(chan struct{})
	go func() { _, _ = io.Copy(io.Discard, r); close(eof) }()
	select {
	case err := <-stop:
		return err
	case <-eof:
		return nil
	}
}

// Conn is the server's end of the protocol. It makes one call at a time.
type Conn struct {
	r    *bufio.Reader
	w    *bufio.Writer
	next uint64

	// OnNotify hears what a helper says unasked, if set before the first call. What it is
	// given is untrusted and may arrive at any rate, so it checks and limits for itself. It
	// runs while the Conn is reading, so it must not block or make a call.
	OnNotify func(method string, params json.RawMessage)

	frames chan frame // set by Listen
}

// frame is one answer from a helper, or the error that ended the reading.
type frame struct {
	res  response
	blob []byte
	err  error
}

// readFrame reads up to the next answer, passing on any notifications before it.
func (c *Conn) readFrame() frame {
	for {
		line, err := readLine(c.r)
		if err != nil {
			return frame{err: err}
		}
		var m struct {
			response
			notification
		}
		if err := json.Unmarshal(line, &m); err != nil {
			return frame{err: fmt.Errorf("unreadable answer: %w", err)}
		}
		if m.Notify != "" {
			if c.OnNotify != nil {
				c.OnNotify(m.Notify, m.notification.Params)
			}
			continue
		}
		if m.Blob < 0 || m.Blob > MaxBlob {
			return frame{err: errors.New("the helper announced an impossible size")}
		}
		f := frame{res: m.response}
		if m.Blob > 0 {
			// Grows with what really arrives, so a false size costs nothing.
			var buf bytes.Buffer
			if _, err := io.CopyN(&buf, c.r, m.Blob); err != nil {
				return frame{err: err}
			}
			f.blob = buf.Bytes()
		}
		return f
	}
}

// Listen keeps reading between calls, for a helper that stays running: without it, nothing
// reads the pipe while no call is being made, and a helper with a lot to say would stop at
// Notify until the next call. It ends when the helper's output does. Call it once, before
// the first call. A Conn that listens can't Stream.
func (c *Conn) Listen() {
	c.frames = make(chan frame, 1)
	go func() {
		for {
			f := c.readFrame()
			// This never blocks. An answer nobody asked for is replaced by the next, and
			// if one is still waiting when a call begins, that call fails on its id.
			select {
			case c.frames <- f:
			default:
				select {
				case <-c.frames:
				default:
				}
				c.frames <- f
			}
			if f.err != nil {
				return
			}
		}
	}()
}

// NewConn wraps a helper's stdout and stdin.
func NewConn(fromHelper io.Reader, toHelper io.Writer) *Conn {
	return &Conn{r: bufio.NewReaderSize(fromHelper, MaxLine), w: bufio.NewWriter(toHelper)}
}

// ReadHello reads what a helper says first.
func (c *Conn) ReadHello() (*Hello, error) {
	line, err := readLine(c.r)
	if err != nil {
		return nil, fmt.Errorf("the helper didn't say hello: %w", err)
	}
	var h Hello
	if err := json.Unmarshal(line, &h); err != nil {
		return nil, fmt.Errorf("the helper's hello is unreadable: %w", err)
	}
	return &h, nil
}

// Call asks the helper one thing and decodes the result into out, when given.
// A *CallError means the helper answered no; any other error means it failed.
func (c *Conn) Call(method string, params, out any) ([]byte, error) {
	c.next++
	req := request{ID: c.next, Method: method}
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		req.Params = b
	}
	if err := writeLine(c.w, req); err != nil {
		return nil, err
	}
	if err := c.w.Flush(); err != nil {
		return nil, err
	}
	var f frame
	if c.frames != nil {
		f = <-c.frames
	} else {
		f = c.readFrame()
	}
	if f.err != nil {
		return nil, f.err
	}
	if f.res.ID != req.ID {
		return nil, errors.New("the helper answered a different call")
	}
	if f.res.Error != "" {
		return nil, &CallError{Code: f.res.Code, Message: f.res.Error}
	}
	if out != nil && len(f.res.Result) > 0 {
		if err := json.Unmarshal(f.res.Result, out); err != nil {
			return nil, fmt.Errorf("unreadable answer: %w", err)
		}
	}
	return f.blob, nil
}

// Stream makes a call that a helper answers with a stream (ServeStream). It returns
// what comes after the answer. Nothing else can be asked on this Conn afterwards.
func (c *Conn) Stream(method string, params, out any) (io.Reader, error) {
	if c.frames != nil {
		return nil, errors.New("a Conn that listens can't stream")
	}
	if _, err := c.Call(method, params, out); err != nil {
		return nil, err
	}
	return c.r, nil
}
