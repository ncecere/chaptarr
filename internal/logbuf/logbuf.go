// Package logbuf is a slog handler that writes to a stream and also keeps
// the most recent records in memory for the UI's log viewer.
package logbuf

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/ncecere/chaptarr/internal/model"
)

type Buffer struct {
	mu      sync.Mutex
	entries []model.LogEntry
	next    int
	full    bool
}

func NewBuffer(size int) *Buffer { return &Buffer{entries: make([]model.LogEntry, size)} }

func (b *Buffer) add(e model.LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries[b.next] = e
	b.next = (b.next + 1) % len(b.entries)
	if b.next == 0 {
		b.full = true
	}
}

// Recent returns up to limit entries, newest first.
func (b *Buffer) Recent(limit int) []model.LogEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := b.next
	if b.full {
		n = len(b.entries)
	}
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]model.LogEntry, 0, limit)
	for i := 1; i <= limit; i++ {
		out = append(out, b.entries[(b.next-i+len(b.entries))%len(b.entries)])
	}
	return out
}

type handler struct {
	inner slog.Handler
	buf   *Buffer
	attrs []slog.Attr
}

// NewLogger logs text to w and mirrors every record into buf.
func NewLogger(w io.Writer, buf *Buffer, level slog.Level) *slog.Logger {
	return slog.New(&handler{inner: slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}), buf: buf})
}

func (h *handler) Enabled(ctx context.Context, l slog.Level) bool { return h.inner.Enabled(ctx, l) }

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	var sb strings.Builder
	sb.WriteString(r.Message)
	write := func(a slog.Attr) bool {
		fmt.Fprintf(&sb, " %s=%v", a.Key, a.Value.Resolve().Any())
		return true
	}
	for _, a := range h.attrs {
		write(a)
	}
	r.Attrs(write)
	h.buf.add(model.LogEntry{Time: r.Time.UTC(), Level: strings.ToLower(r.Level.String()), Message: sb.String()})
	return h.inner.Handle(ctx, r)
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{inner: h.inner.WithAttrs(attrs), buf: h.buf, attrs: append(append([]slog.Attr{}, h.attrs...), attrs...)}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{inner: h.inner.WithGroup(name), buf: h.buf, attrs: h.attrs}
}
