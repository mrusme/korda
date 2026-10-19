package request

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
)

const MaxFieldSize = 1 << 20

var (
	ErrNoFile       = errors.New("no file in the request")
	ErrNotMultipart = errors.New("not a multipart request")
	ErrMultipart    = errs.ErrUnexpectedEndOfForm
)

type partReader struct {
	r io.Reader
}

func (p partReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if err != nil && !errors.Is(err, io.EOF) {
		err = ErrMultipart
	}

	return n, err
}

func basename(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		return name[i+1:]
	}

	return name
}

type Part struct {
	Field    string
	Filename string
	Reader   io.Reader
}

func (req *Request) Parts(each func(p Part) error) error {
	mediaType, params, err := mime.ParseMediaType(req.c.Get(fiber.HeaderContentType))
	if err != nil || mediaType != fiber.MIMEMultipartForm || params["boundary"] == "" {
		return ErrNotMultipart
	}

	mr := multipart.NewReader(req.Stream(), params["boundary"])
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return ErrMultipart
		}

		p := Part{Field: part.FormName(), Filename: basename(part.FileName()), Reader: partReader{part}}
		if p.Filename == "" {
			p.Reader = io.LimitReader(p.Reader, MaxFieldSize+1)
		}
		if err = each(p); err != nil {
			return err
		}
	}
}

func (req *Request) File(field string, accept func(filename string) error, use func(filename string, r io.Reader) error) error {
	used := false
	err := req.Parts(func(p Part) error {
		if p.Filename == "" || used {
			return nil
		}
		if p.Field != field {
			return fmt.Errorf("%w - %s", errs.ErrUnexpectedFileField, p.Field)
		}
		if err := accept(p.Filename); err != nil {
			return err
		}

		buffered := bufio.NewReader(p.Reader)
		if _, err := buffered.Peek(1); errors.Is(err, io.EOF) {
			return errs.ErrFileEmpty
		}

		used = true
		return use(p.Filename, buffered)
	})
	if errors.Is(err, ErrNotMultipart) || (err == nil && !used) {
		return ErrNoFile
	}

	return err
}

func (p Part) Value() (string, error) {
	b, err := io.ReadAll(p.Reader)
	if err != nil {
		return "", err
	}
	if len(b) > MaxFieldSize {
		return "", fmt.Errorf("%w - %s", errs.ErrFieldTooLong, p.Field)
	}

	return string(b), nil
}

func (req *Request) Drain() {
	if _, err := io.Copy(io.Discard, req.Stream()); err != nil {
		req.rt.Debug("request", "drain", "error", err)
	}
}

func (req *Request) SendFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}

	if err := req.c.SendFile(path, fiber.SendFile{ByteRange: true}); err != nil {
		return err
	}
	req.c.Set(fiber.HeaderCacheControl, "public, max-age=0")

	return nil
}
