package request

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
)

var ErrNoFile = errors.New("no file in the request")

func (req *Request) File(field string, accept func(filename string) error, use func(filename string, r io.Reader) error) error {
	mediaType, params, err := mime.ParseMediaType(req.c.Get(fiber.HeaderContentType))
	if err != nil || mediaType != fiber.MIMEMultipartForm || params["boundary"] == "" {
		return ErrNoFile
	}

	mr := multipart.NewReader(req.Stream(), params["boundary"])
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return ErrNoFile
		}
		if err != nil {
			return err
		}

		if part.FileName() == "" {
			continue
		}
		if part.FormName() != field {
			return fmt.Errorf("%w - %s", errs.ErrUnexpectedFileField, part.FormName())
		}
		if err = accept(part.FileName()); err != nil {
			return err
		}

		buffered := bufio.NewReader(part)
		if _, err = buffered.Peek(1); errors.Is(err, io.EOF) {
			return errs.ErrFileEmpty
		}

		return use(part.FileName(), buffered)
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
