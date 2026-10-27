package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/services/mediaroot"
)

func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.NRGBA{R: 200, G: 40, B: 90, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestOutsideIsSharpsOutsideFitWithoutEnlargement(t *testing.T) {
	tests := []struct{ w, h, wantW, wantH int }{
		{400, 300, 333, 250},
		{300, 400, 250, 333},
		{250, 250, 250, 250},
		{200, 120, 200, 120},
		{4032, 3024, 333, 250},
	}

	for _, tt := range tests {
		got := Outside(image.NewNRGBA(image.Rect(0, 0, tt.w, tt.h)), 250).Bounds()
		if got.Dx() != tt.wantW || got.Dy() != tt.wantH {
			t.Errorf("Outside(%dx%d) = %dx%d, want %dx%d", tt.w, tt.h, got.Dx(), got.Dy(), tt.wantW, tt.wantH)
		}
	}
}

func TestThumbnailWritesAJPEG(t *testing.T) {
	root, err := mediaroot.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, _ := New(root, 250, 80)

	if err = m.Thumbnail(bytes.NewReader(pngOf(t, 400, 300)), "profile/u/avatar.jpg"); err != nil {
		t.Fatal(err)
	}

	abs, _ := root.Abs("profile/u/avatar.jpg")
	f, err := os.Open(abs)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil || img.Bounds().Dx() != 333 || img.Bounds().Dy() != 250 {
		t.Errorf("the written image is %v, %v, want a JPEG of 333x250", img.Bounds(), err)
	}
}

func TestThumbnailRejectsWhatItCannotDecode(t *testing.T) {
	root, _ := mediaroot.New(t.TempDir())
	m, _ := New(root, 250, 80)

	err := m.Thumbnail(bytes.NewReader([]byte("not an image")), "profile/u/x.jpg")
	if !errors.Is(err, errs.ErrImageUndecodable) {
		t.Errorf("Thumbnail of text = %v, want %v", err, errs.ErrImageUndecodable)
	}
}

func fakeFFmpeg(t *testing.T, decoders string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "ffmpeg")
	script := "#!/bin/sh\ncase \"$2\" in\n" +
		"-encoders) printf ' V....D libx264              libx264 H.264\\n' ;;\n" +
		"-decoders) printf '%s' '" + decoders + "' ;;\n" +
		"-filters) printf ' ... zscale            V->V       Apply resizing\\n ... tonemap           V->V       Conversion\\n' ;;\n" +
		"esac\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestDetectFFmpegHEVC(t *testing.T) {
	cases := []struct {
		name     string
		decoders string
		want     bool
	}{
		{"software decoder", " VFS..D hevc                 HEVC (High Efficiency Video Coding)\n V....D h264                 H.264\n", true},
		{"hardware decoders only", " V....D hevc_qsv             HEVC video (Intel Quick Sync Video acceleration) (codec hevc)\n V..... hevc_v4l2m2m         V4L2 mem2mem HEVC decoder wrapper (codec hevc)\n", false},
		{"no decoder", " VFS..D h264                 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := DetectFFmpeg(context.Background(), fakeFFmpeg(t, c.decoders))
			if f == nil {
				t.Fatal("DetectFFmpeg = nil")
			}
			if f.HEVC != c.want || f.Encoder != EncoderX264 || !f.ToneMap {
				t.Errorf("DetectFFmpeg = HEVC %t, encoder %q, tone mapping %t, want HEVC %t", f.HEVC, f.Encoder, f.ToneMap, c.want)
			}
		})
	}
}
