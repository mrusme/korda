package compress

import (
	"bufio"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"sync"

	"github.com/andybalholm/brotli"
)

const (
	bufferSize     = 16 << 10
	brotliQuality  = 4
	brotliWindow   = 22
	identityMethod = "identity"
)

type Body struct {
	write    func(w io.Writer) error
	encoding string
	once     sync.Once
	pipe     *io.PipeReader
}

func Stream(write func(w io.Writer) error) *Body {
	return &Body{write: write}
}

func Bytes(b []byte) *Body {
	return Stream(func(w io.Writer) error {
		_, err := w.Write(b)
		return err
	})
}

func (b *Body) SetEncoding(encoding string) {
	b.encoding = encoding
}

func (b *Body) SupportsBodyWriteTo() bool {
	return true
}

type counter struct {
	w io.Writer
	n int64
}

func (c *counter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)

	return n, err
}

func encoder(w io.Writer, encoding string) (io.WriteCloser, error) {
	switch encoding {
	case "gzip":
		return gzip.NewWriter(w), nil
	case "br":
		return brotli.NewWriterOptions(w, brotli.WriterOptions{Quality: brotliQuality, LGWin: brotliWindow}), nil
	case "deflate":
		return zlib.NewWriter(w), nil
	}

	return nil, fmt.Errorf("unsupported encoding %q", encoding)
}

func (b *Body) WriteTo(w io.Writer) (int64, error) {
	out := &counter{w: w}
	if b.encoding == "" || b.encoding == identityMethod {
		err := b.write(out)
		return out.n, err
	}

	buf := bufio.NewWriterSize(out, bufferSize)
	enc, err := encoder(buf, b.encoding)
	if err != nil {
		return 0, err
	}
	if err = b.write(enc); err != nil {
		return out.n, err
	}
	if err = enc.Close(); err != nil {
		return out.n, err
	}
	err = buf.Flush()

	return out.n, err
}

func (b *Body) Read(p []byte) (int, error) {
	b.once.Do(func() {
		r, w := io.Pipe()
		b.pipe = r
		go func() {
			_, err := b.WriteTo(w)
			_ = w.CloseWithError(err)
		}()
	})

	return b.pipe.Read(p)
}

func (b *Body) Close() error {
	if b.pipe != nil {
		return b.pipe.Close()
	}

	return nil
}
