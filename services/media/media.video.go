package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"tty.fail/mrus/korda/services/metadata/video"
)

var ErrNoFFmpeg = errors.New("ffmpeg is not available")

const (
	EncoderX264     = "libx264"
	EncoderOpenH264 = "libopenh264"

	thumbnailFilters = `fps=12:start_time=0:eof_action=pass:round=down,thumbnail=12,select=gt(scene\,0.1)-eq(prev_selected_n\,n)+isnan(prev_selected_n)+gt(n\,20),trim=end_frame=2,reverse`
	toneMapFilters   = "zscale=t=linear:npl=100,format=gbrpf32le,zscale=p=bt709,tonemap=tonemap=hable:desat=0,zscale=t=bt709:m=bt709:r=pc,format=yuv420p"
)

type FFmpeg struct {
	Path    string
	Encoder string
	ToneMap bool
}

func DetectFFmpeg(ctx context.Context, path string) *FFmpeg {
	if path == "" {
		return nil
	}

	run := func(args ...string) string {
		out, err := exec.CommandContext(ctx, path, append([]string{"-hide_banner"}, args...)...).Output()
		if err != nil {
			return ""
		}
		return string(out)
	}

	encoders := run("-encoders")
	if encoders == "" {
		return nil
	}

	f := &FFmpeg{Path: path}
	switch {
	case strings.Contains(encoders, " "+EncoderX264+" "):
		f.Encoder = EncoderX264
	case strings.Contains(encoders, " "+EncoderOpenH264+" "):
		f.Encoder = EncoderOpenH264
	}

	filters := run("-filters")
	f.ToneMap = strings.Contains(filters, " zscale ") && strings.Contains(filters, " tonemap ")

	return f
}

func rotated(s *video.Stream) bool {
	return s.Rotation == 90 || s.Rotation == -90 || s.Rotation == 270 || s.Rotation == -270
}

func Vertical(s *video.Stream) bool {
	return (s.Height > s.Width) != rotated(s)
}

func target(size int) int {
	if size%2 != 0 {
		size--
	}

	return size
}

func shouldScale(s *video.Stream, size int) bool {
	odd := s.Height%2 != 0 || s.Width%2 != 0
	return odd || min(s.Height, s.Width) > target(size)
}

func scaling(s *video.Stream, size int) string {
	if Vertical(s) {
		return strconv.Itoa(target(size)) + ":-2"
	}

	return "-2:" + strconv.Itoa(target(size))
}

func hdr(s *video.Stream) bool {
	return s.ColorTransfer == "smpte2084" || s.ColorTransfer == "arib-std-b67"
}

func (f *FFmpeg) colorFilters(s *video.Stream) []string {
	if hdr(s) && f.ToneMap {
		return []string{toneMapFilters}
	}
	if !strings.HasSuffix(s.PixelFormat, "420p") {
		return []string{"format=yuv420p"}
	}

	return nil
}

func (f *FFmpeg) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, f.Path, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(lastLine(stderr.String())))
	}

	return nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}

func (f *FFmpeg) Frame(ctx context.Context, input string, info *video.Info, size int, output string) error {
	if f == nil {
		return ErrNoFFmpeg
	}
	s := info.Video
	if s == nil {
		return fmt.Errorf("%w: no video stream", ErrNoFFmpeg)
	}

	var args []string
	if info.FormatName != "mpegts" {
		args = append(args, "-skip_frame", "nointra")
	}
	args = append(args, "-sws_flags", "accurate_rnd+full_chroma_int", "-i", input, "-y",
		"-fps_mode", "vfr", "-frames:v", "1", "-update", "1", "-v", "error")

	filters := []string{thumbnailFilters}
	if shouldScale(s, size) {
		filters = append(filters, "scale="+scaling(s, size)+":flags=lanczos+accurate_rnd+full_chroma_int:out_range=pc")
	}
	filters = append(filters, f.colorFilters(s)...)
	args = append(args, "-vf", strings.Join(filters, ","), "-q:v", "2", output)

	return f.run(ctx, args...)
}

type TranscodeAction int

const (
	TranscodeNone TranscodeAction = iota
	TranscodeRemux
	TranscodeAudio
	TranscodeVideo
	TranscodeAll
)

var acceptedContainers = []string{"mov", "mp4", "ogg", "webm"}

func container(info *video.Info) string {
	switch info.FormatLongName {
	case "QuickTime / MOV":
		return "mov"
	case "Matroska / WebM":
		return "webm"
	}

	return info.FormatName
}

func Decide(info *video.Info) TranscodeAction {
	audio := info.Audio != nil && info.Audio.CodecName != "aac" && info.Audio.CodecName != "mp3" && info.Audio.CodecName != "opus"
	videoNeeded := info.Video != nil && (info.Video.CodecName != "h264" || !strings.HasSuffix(info.Video.PixelFormat, "420p"))

	switch {
	case audio && videoNeeded:
		return TranscodeAll
	case videoNeeded:
		return TranscodeVideo
	case audio:
		return TranscodeAudio
	}

	if slices.Contains(acceptedContainers, container(info)) {
		return TranscodeNone
	}

	return TranscodeRemux
}

func (f *FFmpeg) Transcode(ctx context.Context, input string, info *video.Info, resolution int, output string) error {
	if f == nil {
		return ErrNoFFmpeg
	}
	action := Decide(info)
	if action == TranscodeNone {
		return nil
	}
	if (action == TranscodeVideo || action == TranscodeAll) && f.Encoder == "" {
		return fmt.Errorf("%w: no H.264 encoder", ErrNoFFmpeg)
	}

	s := info.Video
	args := []string{"-i", input, "-y"}

	switch action {
	case TranscodeVideo, TranscodeAll:
		args = append(args, "-c:v", f.Encoder)
	default:
		args = append(args, "-c:v", "copy")
	}
	switch action {
	case TranscodeAudio, TranscodeAll:
		args = append(args, "-c:a", "aac")
	default:
		args = append(args, "-c:a", "copy")
	}

	args = append(args, "-map", "0:"+strconv.Itoa(s.Index), "-map_metadata", "-1")
	if info.Audio != nil {
		args = append(args, "-map", "0:"+strconv.Itoa(info.Audio.Index))
		if action == TranscodeAudio || action == TranscodeAll {
			args = append(args, "-ac", "2")
		}
	}

	if action == TranscodeVideo || action == TranscodeAll {
		switch f.Encoder {
		case EncoderX264:
			args = append(args, "-preset", "ultrafast", "-crf", "23")
		case EncoderOpenH264:
			args = append(args, "-b:v", openH264Bitrate(s, resolution))
		}
	}
	args = append(args, "-movflags", "faststart", "-fps_mode", "passthrough", "-v", "error")

	if action == TranscodeVideo || action == TranscodeAll {
		var filters []string
		if shouldScale(s, resolution) {
			filters = append(filters, "scale="+scaling(s, resolution))
		}
		filters = append(filters, f.colorFilters(s)...)
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o750); err != nil {
		return err
	}
	tmp := output + ".part.mp4"
	if err := f.run(ctx, append(args, "-f", "mp4", tmp)...); err != nil {
		os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, output)
}

func openH264Bitrate(s *video.Stream, resolution int) string {
	short := min(min(s.Width, s.Height), resolution)
	kbps := int(math.Max(800, float64(short)*float64(short)*16/9*2.5/1000))

	return strconv.Itoa(kbps) + "k"
}
