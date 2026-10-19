package video

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var ErrNoProbe = errors.New("ffprobe is not available")

type Stream struct {
	Index         int
	CodecType     string
	CodecName     string
	Profile       string
	Width         int
	Height        int
	PixelFormat   string
	FrameRate     float64
	BitRate       int64
	Rotation      int
	DurationSecs  float64
	CreationTime  string
	ColorTransfer string
	ColorPrimary  string
	ColorSpace    string
	TimeBase      string
	Default       bool
	AttachedPic   bool
}

type Info struct {
	FormatName     string
	FormatLongName string
	Duration       time.Duration
	BitRate        int64
	CreatedAt      time.Time
	Tags           map[string]string
	Video          *Stream
	Audio          *Stream
	Streams        []Stream
}

type probeJSON struct {
	Streams []struct {
		Index          int               `json:"index"`
		CodecType      string            `json:"codec_type"`
		CodecName      string            `json:"codec_name"`
		Profile        string            `json:"profile"`
		Width          int               `json:"width"`
		Height         int               `json:"height"`
		PixFmt         string            `json:"pix_fmt"`
		AvgFrameRate   *string           `json:"avg_frame_rate"`
		RFrameRate     string            `json:"r_frame_rate"`
		BitRate        string            `json:"bit_rate"`
		Duration       string            `json:"duration"`
		DAR            string            `json:"display_aspect_ratio"`
		ColorTransfer  string            `json:"color_transfer"`
		ColorPrimaries string            `json:"color_primaries"`
		ColorSpace     string            `json:"color_space"`
		TimeBase       string            `json:"time_base"`
		Tags           map[string]string `json:"tags"`
		SideData       []struct {
			Rotation *float64 `json:"rotation"`
		} `json:"side_data_list"`
		Disposition map[string]int `json:"disposition"`
	} `json:"streams"`
	Format struct {
		FormatName     string            `json:"format_name"`
		FormatLongName string            `json:"format_long_name"`
		Duration       string            `json:"duration"`
		BitRate        string            `json:"bit_rate"`
		Tags           map[string]string `json:"tags"`
	} `json:"format"`
}

func rate(s string) float64 {
	num, den, ok := strings.Cut(s, "/")
	if !ok {
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}
	n, err1 := strconv.ParseFloat(num, 64)
	d, err2 := strconv.ParseFloat(den, 64)
	if err1 != nil || err2 != nil || d == 0 {
		return 0
	}

	return n / d
}

func Probe(ctx context.Context, ffprobe string, path string) (*Info, error) {
	if ffprobe == "" {
		return nil, ErrNoProbe
	}

	cmd := exec.CommandContext(ctx, ffprobe, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, ErrNoProbe
		}
		return nil, err
	}

	var parsed probeJSON
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		return nil, err
	}

	info := &Info{FormatName: parsed.Format.FormatName, FormatLongName: parsed.Format.FormatLongName, Tags: parsed.Format.Tags}
	if info.Tags == nil {
		info.Tags = map[string]string{}
	}
	if secs, err := strconv.ParseFloat(parsed.Format.Duration, 64); err == nil && !math.IsNaN(secs) {
		info.Duration = time.Duration(secs * float64(time.Second))
	}
	info.BitRate, _ = strconv.ParseInt(parsed.Format.BitRate, 10, 64)
	if raw := parsed.Format.Tags["creation_time"]; raw != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			info.CreatedAt = t.UTC()
		}
	}

	for _, s := range parsed.Streams {
		st := Stream{
			Index:         s.Index,
			CodecType:     s.CodecType,
			CodecName:     s.CodecName,
			Profile:       s.Profile,
			Width:         s.Width,
			Height:        s.Height,
			PixelFormat:   s.PixFmt,
			ColorTransfer: s.ColorTransfer,
			ColorPrimary:  s.ColorPrimaries,
			ColorSpace:    s.ColorSpace,
			TimeBase:      s.TimeBase,
			CreationTime:  s.Tags["creation_time"],
			Default:       s.Disposition["default"] == 1,
			AttachedPic:   s.Disposition["attached_pic"] == 1,
		}
		if st.CodecName == "h265" {
			st.CodecName = "hevc"
		}
		if st.PixelFormat == "" {
			st.PixelFormat = "yuv420p"
		}
		if s.AvgFrameRate != nil {
			st.FrameRate = rate(*s.AvgFrameRate)
		} else {
			st.FrameRate = rate(s.RFrameRate)
		}
		if dar := ratio(s.DAR); dar > 0 && st.Height > 0 {
			st.Width = int(math.Round(float64(st.Height) * dar))
		}
		st.BitRate, _ = strconv.ParseInt(s.BitRate, 10, 64)
		st.DurationSecs, _ = strconv.ParseFloat(s.Duration, 64)
		for _, side := range s.SideData {
			if side.Rotation != nil {
				st.Rotation = int(math.Round(*side.Rotation))
			}
		}
		if st.Rotation == 0 {
			if r, err := strconv.Atoi(s.Tags["rotate"]); err == nil {
				st.Rotation = -r
			}
		}
		info.Streams = append(info.Streams, st)
	}

	info.Video = best(info.Streams, func(s *Stream) bool { return s.CodecType == "video" && !s.AttachedPic })
	info.Audio = best(info.Streams, func(s *Stream) bool { return s.CodecType == "audio" && s.CodecName != "unknown" })

	return info, nil
}

func ratio(s string) float64 {
	w, h, ok := strings.Cut(s, ":")
	if !ok {
		return 0
	}
	a, err1 := strconv.ParseFloat(w, 64)
	b, err2 := strconv.ParseFloat(h, 64)
	if err1 != nil || err2 != nil || a == 0 || b == 0 {
		return 0
	}

	return a / b
}

func best(streams []Stream, keep func(s *Stream) bool) *Stream {
	var out *Stream
	for i := range streams {
		s := &streams[i]
		if !keep(s) {
			continue
		}
		switch {
		case out == nil:
			out = s
		case s.Default && !out.Default:
			out = s
		case s.Default == out.Default && s.BitRate > out.BitRate:
			out = s
		}
	}

	return out
}
