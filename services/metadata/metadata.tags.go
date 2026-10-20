package metadata

import (
	"context"
	"errors"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"tty.fail/mrus/korda/helpers/mimetypes"
	"tty.fail/mrus/korda/services/metadata/exif"
	"tty.fail/mrus/korda/services/metadata/probe"
	"tty.fail/mrus/korda/services/metadata/video"
	"tty.fail/mrus/korda/services/metadata/xmp"
)

var DateTags = []string{
	"SubSecDateTimeOriginal", "SubSecCreateDate", "DateTimeOriginal", "CreationDate", "CreateDate",
	"MediaCreateDate", "DateTimeCreated", "GPSDateTime", "DateTimeUTC", "SonyDateTime2", "SourceImageCreateTime",
}

var capturedAtTags = []string{
	"SubSecDateTimeOriginal", "SubSecCreateDate", "DateTimeOriginal", "CreateDate", "MediaCreateDate",
	"CreationDate", "DateTimeCreated", "TimeCreated",
}

type Tags struct {
	Dates             map[string]Date
	Zone              *int
	TZ                string
	ZoneSource        string
	Make              string
	Model             string
	LensModel         string
	Description       string
	ExposureTime      string
	ProjectionType    string
	ColorSpace        string
	ContentIdentifier string
	BurstID           string
	Orientation       *int
	Width             int
	Height            int
	Latitude          *float64
	Longitude         *float64
	FNumber           *float64
	FocalLength       *float64
	ISO               *int64
	Rating            *float64
	BitsPerSample     *int64
	Duration          *float64
	FrameRate         *float64
	Keywords          []string
	MotionPhoto       bool
	MicroVideo        bool
	MicroVideoOffset  int64
	Directory         []xmp.Item
	IsVideo           bool
	Probe             *probe.Info
	Video             *video.Info
}

func newTags() *Tags {
	return &Tags{Dates: make(map[string]Date)}
}

func (t *Tags) Instant(d Date) time.Time {
	if d.Offset == nil && t.TZ != "" {
		if loc, err := time.LoadLocation(t.TZ); err == nil {
			w := d.Wall
			return time.Date(w.Year(), w.Month(), w.Day(), w.Hour(), w.Minute(), w.Second(), w.Nanosecond(), loc)
		}
	}

	return d.Instant(t.Zone)
}

func (t *Tags) FirstDate() (Date, bool) {
	for _, tag := range DateTags {
		if d, ok := t.Dates[tag]; ok {
			return d, true
		}
	}

	return Date{}, false
}

func (t *Tags) setDate(tag string, raw string) {
	if _, exists := t.Dates[tag]; exists {
		return
	}
	if d, ok := ParseDate(tag, raw); ok {
		t.Dates[tag] = d
	}
}

func trimEXIF(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "\x00 ")
}

func setString(dst *string, value string) {
	if *dst == "" {
		*dst = trimEXIF(value)
	}
}

func round6(f float64) float64 {
	return math.Round(f*1e6) / 1e6
}

func exposure(secs float64) string {
	if secs > 0 && secs < 0.25001 {
		return "1/" + strconv.Itoa(int(0.5+1/secs))
	}

	s := strconv.FormatFloat(secs, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0")
}

func oneDecimal(f float64) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(f, 'f', 1, 64), 64)
	return v
}

func formatNumber(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

var colorSpaces = map[uint64]string{1: "sRGB", 2: "Adobe RGB", 0xfffd: "Wide Gamut RGB", 0xfffe: "ICC Profile", 0xffff: "Uncalibrated"}

func degrees(e exif.Entry) (float64, bool) {
	var total float64
	for i, div := range []float64{1, 60, 3600} {
		num, den, ok := e.Rational(i)
		if !ok {
			if i == 0 {
				return 0, false
			}
			break
		}
		if den == 0 {
			continue
		}
		total += float64(num) / float64(den) / div
	}

	return total, true
}

func (t *Tags) applyEXIF(tf *exif.TIFF) {
	ifd0, ex, gps := tf.IFD0, tf.Exif, tf.GPS

	if e, ok := ifd0[0x010f]; ok {
		setString(&t.Make, e.String())
	}
	if e, ok := ifd0[0x0110]; ok {
		setString(&t.Model, e.String())
	}
	if e, ok := ifd0[0x010e]; ok {
		setString(&t.Description, e.String())
	}
	if e, ok := ifd0[0x0112]; ok && t.Orientation == nil {
		if v, ok := e.Uint(0); ok {
			o := int(v)
			t.Orientation = &o
		}
	}
	if e, ok := ifd0[0x4746]; ok && t.Rating == nil {
		if v, ok := e.Float(0); ok {
			t.Rating = &v
		}
	}
	if e, ok := ifd0[0x0102]; ok && t.BitsPerSample == nil {
		if v, ok := e.Uint(0); ok {
			n := int64(v)
			t.BitsPerSample = &n
		}
	}

	if e, ok := ex[0x829a]; ok && t.ExposureTime == "" {
		if v, ok := e.Float(0); ok {
			t.ExposureTime = exposure(v)
		}
	}
	if e, ok := ex[0x829d]; ok && t.FNumber == nil {
		if v, ok := e.Float(0); ok && !math.IsNaN(v) && !math.IsInf(v, 0) {
			f := oneDecimal(v)
			t.FNumber = &f
		}
	}
	if e, ok := ex[0x920a]; ok && t.FocalLength == nil {
		if v, ok := e.Float(0); ok && !math.IsNaN(v) && !math.IsInf(v, 0) {
			t.FocalLength = &v
		}
	}
	if t.ISO == nil {
		for _, tag := range []uint16{0x8832, 0x8831, 0x8827} {
			if e, ok := ex[tag]; ok {
				if v, ok := e.Uint(0); ok && v != 0 {
					n := int64(v)
					t.ISO = &n
					break
				}
			}
		}
	}
	if e, ok := ex[0xa434]; ok {
		setString(&t.LensModel, e.String())
	}
	if t.LensModel == "" {
		if e, ok := ex[0xa432]; ok {
			t.LensModel = lensSpec(e)
		}
	}
	if e, ok := ex[0x927c]; ok {
		if apple := exif.AppleMakerNote(e.Bytes()); apple != nil {
			if id := trimEXIF(apple[0x0011].String()); id != "" && t.ContentIdentifier == "" {
				t.ContentIdentifier = id
			}
			if id := trimEXIF(apple[0x000b].String()); id != "" && t.BurstID == "" {
				t.BurstID = id
			}
		}
	}
	if e, ok := ex[0xa001]; ok && t.ColorSpace == "" {
		if v, ok := e.Uint(0); ok {
			if name, known := colorSpaces[v]; known {
				t.ColorSpace = name
			} else {
				t.ColorSpace = strconv.FormatUint(v, 10)
			}
		}
	}

	subsec := func(tag uint16) string {
		if e, ok := ex[tag]; ok {
			return strings.TrimSpace(e.String())
		}
		return ""
	}
	offset := func(tag uint16) string {
		if e, ok := ex[tag]; ok {
			return strings.TrimSpace(e.String())
		}
		return ""
	}

	if e, ok := ex[0x9003]; ok {
		raw := trimEXIF(e.String())
		t.setDate("DateTimeOriginal", raw)
		composite := raw
		if s := subsec(0x9291); s != "" {
			composite += "." + s
		}
		composite += offset(0x9011)
		t.setDate("SubSecDateTimeOriginal", composite)
	}
	if e, ok := ex[0x9004]; ok {
		raw := trimEXIF(e.String())
		t.setDate("CreateDate", raw)
		composite := raw
		if s := subsec(0x9292); s != "" {
			composite += "." + s
		}
		composite += offset(0x9012)
		t.setDate("SubSecCreateDate", composite)
	}

	if t.Zone == nil {
		for _, tag := range []uint16{0x9011, 0x9012} {
			if minutes, ok := ParseOffset(offset(tag)); ok {
				t.Zone, t.ZoneSource = &minutes, "offset"
				break
			}
		}
	}
	if t.Zone == nil {
		if e, ok := ex[0x882a]; ok {
			if hours, ok := e.Int(0); ok && hours >= -12 && hours <= 14 {
				minutes := int(hours) * 60
				t.Zone, t.ZoneSource = &minutes, "offset"
			}
		}
	}

	if len(gps) > 0 && t.Latitude == nil {
		lat, okLat := degrees(gps[0x0002])
		lon, okLon := degrees(gps[0x0004])
		if okLat && okLon {
			if ref := strings.ToUpper(strings.TrimSpace(gps[0x0001].String())); strings.HasPrefix(ref, "S") {
				lat = -math.Abs(lat)
			}
			if ref := strings.ToUpper(strings.TrimSpace(gps[0x0003].String())); strings.HasPrefix(ref, "W") {
				lon = -math.Abs(lon)
			}
			t.setGPS(lat, lon)
		}

		stamp := trimEXIF(gps[0x001d].String())
		if h, okH := degrees(gps[0x0007]); okH && stamp != "" {
			secs := int(math.Round(h * 3600))
			t.setDate("GPSDateTime", stamp+" "+strconv.Itoa(secs/3600)+":"+strconv.Itoa(secs/60%60)+":"+strconv.Itoa(secs%60)+"Z")
		}
	}
}

func (t *Tags) setGPS(lat float64, lon float64) {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.Abs(lat) > 90 || math.Abs(lon) > 180 || (lat == 0 && lon == 0) {
		return
	}

	lat, lon = round6(lat), round6(lon)
	t.Latitude, t.Longitude = &lat, &lon
}

func lensSpec(e exif.Entry) string {
	var values []float64
	for i := range 4 {
		num, den, ok := e.Rational(i)
		if !ok || den == 0 {
			values = append(values, 0)
			continue
		}
		values = append(values, float64(num)/float64(den))
	}
	if len(values) < 4 || values[0] == 0 {
		return ""
	}

	s := formatNumber(values[0])
	if values[1] != 0 && values[1] != values[0] {
		s += "-" + formatNumber(values[1])
	}
	s += "mm f/" + formatNumber(values[2])
	if values[3] != 0 && values[3] != values[2] {
		s += "-" + formatNumber(values[3])
	}

	return s
}

var xmpGPS = regexp.MustCompile(`^(\d+),(\d+(?:\.\d+)?)(?:,(\d+(?:\.\d+)?))?([NSEW])$`)

func xmpCoordinate(s string) (float64, bool) {
	m := xmpGPS.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return f, err == nil
	}

	deg, _ := strconv.ParseFloat(m[1], 64)
	minutes, _ := strconv.ParseFloat(m[2], 64)
	seconds, _ := strconv.ParseFloat(m[3], 64)
	v := deg + minutes/60 + seconds/3600
	if m[4] == "S" || m[4] == "W" {
		v = -v
	}

	return v, true
}

func (t *Tags) applyXMP(p *xmp.Packet) {
	for _, d := range []struct {
		tag   string
		space string
		local string
	}{
		{"DateTimeOriginal", xmp.NSEXIF, "DateTimeOriginal"},
		{"CreateDate", xmp.NSXMP, "CreateDate"},
		{"DateTimeCreated", xmp.NSPhotoshop, "DateCreated"},
		{"SourceImageCreateTime", xmp.NSGPano, "SourceImageCreateTime"},
	} {
		if v, ok := p.String(d.space, d.local); ok {
			t.setDate(d.tag, v)
		}
	}

	if t.Description == "" {
		if v, ok := p.String(xmp.NSDC, "description"); ok {
			t.Description = strings.TrimSpace(v)
		}
	}
	if v, ok := p.Float(xmp.NSXMP, "Rating"); ok {
		t.Rating = &v
	}
	if t.Make == "" {
		if v, ok := p.String(xmp.NSTIFF, "Make"); ok {
			t.Make = trimEXIF(v)
		}
	}
	if t.Model == "" {
		if v, ok := p.String(xmp.NSTIFF, "Model"); ok {
			t.Model = trimEXIF(v)
		}
	}
	if t.LensModel == "" {
		for _, space := range []string{xmp.NSEXIFEX, xmp.NSEXIF} {
			if v, ok := p.String(space, "LensModel"); ok {
				t.LensModel = strings.TrimSpace(v)
				break
			}
		}
	}
	if t.Orientation == nil {
		if v, ok := p.Int(xmp.NSTIFF, "Orientation"); ok {
			o := int(v)
			t.Orientation = &o
		}
	}
	if t.Latitude == nil {
		latRaw, okLat := p.String(xmp.NSEXIF, "GPSLatitude")
		lonRaw, okLon := p.String(xmp.NSEXIF, "GPSLongitude")
		if okLat && okLon {
			lat, ok1 := xmpCoordinate(latRaw)
			lon, ok2 := xmpCoordinate(lonRaw)
			if ok1 && ok2 {
				t.setGPS(lat, lon)
			}
		}
	}
	if v, ok := p.String(xmp.NSGPano, "ProjectionType"); ok && t.ProjectionType == "" {
		t.ProjectionType = strings.ToUpper(strings.TrimSpace(v))
	}

	if t.Keywords == nil {
		switch {
		case hasList(p, xmp.NSDigiKam, "TagsList"):
			t.Keywords, _ = p.List(xmp.NSDigiKam, "TagsList")
		case hasList(p, xmp.NSLightroom, "hierarchicalSubject"):
			list, _ := p.List(xmp.NSLightroom, "hierarchicalSubject")
			for _, tag := range list {
				parts := strings.Split(tag, "|")
				for i, part := range parts {
					parts[i] = strings.ReplaceAll(part, "/", "|")
				}
				t.Keywords = append(t.Keywords, strings.Join(parts, "/"))
			}
		case hasList(p, xmp.NSDC, "subject"):
			t.Keywords, _ = p.List(xmp.NSDC, "subject")
		}
	}

	if v, ok := p.Int(xmp.NSGCamera, "MotionPhoto"); ok && v == 1 {
		t.MotionPhoto = true
	}
	if v, ok := p.Int(xmp.NSGCamera, "MicroVideo"); ok && v == 1 {
		t.MicroVideo = true
	}
	if v, ok := p.Int(xmp.NSGCamera, "MicroVideoOffset"); ok {
		t.MicroVideoOffset = v
	}
	if items := p.Directory(); len(items) > 0 {
		t.Directory = items
	}
}

func hasList(p *xmp.Packet, space string, local string) bool {
	_, ok := p.List(space, local)
	return ok
}

var iso6709 = regexp.MustCompile(`^([+-]\d+(?:\.\d+)?)([+-]\d+(?:\.\d+)?)`)

func (t *Tags) applyVideo(v *video.Info) {
	t.Video = v
	if v.Video != nil {
		if v.Video.Width > 0 {
			t.Width = v.Video.Width
		}
		if v.Video.Height > 0 {
			t.Height = v.Video.Height
		}

		var o int
		switch v.Video.Rotation {
		case -90:
			o = 6
		case 0:
			o = 1
		case 90:
			o = 8
		case 180:
			o = 3
		}
		if o != 0 {
			t.Orientation = &o
		}
		if v.Video.FrameRate > 0 {
			fps := v.Video.FrameRate
			t.FrameRate = &fps
		}
	}
	if secs := v.Duration.Seconds(); secs > 0 {
		t.Duration = &secs
	}

	tags := v.Tags
	if raw := tags["com.apple.quicktime.creationdate"]; raw != "" {
		t.setDate("CreationDate", raw)
	}
	if raw := tags["creation_time"]; raw != "" {
		t.setDate("CreateDate", strings.TrimSuffix(raw, "Z"))
	}
	if v.Video != nil {
		if raw := v.Video.CreationTime; raw != "" {
			t.setDate("MediaCreateDate", strings.TrimSuffix(raw, "Z"))
		}
	}

	setString(&t.Make, tags["com.apple.quicktime.make"])
	setString(&t.Make, tags["com.android.manufacturer"])
	setString(&t.Model, tags["com.apple.quicktime.model"])
	setString(&t.Model, tags["com.android.model"])
	if id := tags["com.apple.quicktime.content.identifier"]; id != "" && t.ContentIdentifier == "" {
		t.ContentIdentifier = id
	}

	for _, key := range []string{"com.apple.quicktime.location.ISO6709", "location"} {
		if m := iso6709.FindStringSubmatch(tags[key]); m != nil && t.Latitude == nil {
			lat, _ := strconv.ParseFloat(m[1], 64)
			lon, _ := strconv.ParseFloat(m[2], 64)
			t.setGPS(lat, lon)
		}
	}
}

func (t *Tags) inferZone() {
	if t.Zone != nil || t.TZ != "" {
		return
	}

	if t.Latitude != nil && t.Longitude != nil {
		if name, ok := gpsZone(*t.Latitude, *t.Longitude); ok {
			t.TZ, t.ZoneSource = name, "gps"
			return
		}
	}

	for _, tag := range capturedAtTags {
		if d, ok := t.Dates[tag]; ok && d.Offset != nil && *d.Offset != 0 {
			offset := *d.Offset
			t.Zone, t.ZoneSource = &offset, "datestamp"
			return
		}
	}

	if t.IsVideo {
		utc := 0
		t.Zone, t.ZoneSource = &utc, "video"
		return
	}

	gps, ok := t.Dates["GPSDateTime"]
	if !ok {
		return
	}
	for _, tag := range capturedAtTags {
		d, ok := t.Dates[tag]
		if !ok || d.Offset != nil {
			continue
		}
		if offset, ok := inferLikelyOffset(d.Wall.Sub(gps.Wall).Minutes()); ok {
			t.Zone, t.ZoneSource = &offset, "gps-time"
		}
		return
	}
}

func Read(ctx context.Context, path string, ffprobe string, kind string) (*Tags, error) {
	t := newTags()
	t.IsVideo = kind == "VIDEO"

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}

	info, err := probe.Probe(f, st.Size())
	if err != nil && info == nil {
		return nil, err
	}
	t.Probe = info

	var tiff *exif.TIFF
	switch {
	case info.EXIF != nil:
		tiff, _ = exif.Parse(info.EXIF)
	case info.CR3Exif != nil:
		tiff, _ = exif.Parse(info.CR3Exif)
	}
	if tiff != nil {
		t.applyEXIF(tiff)
		t.dimensionsFromTIFF(tiff)
	}
	if info.XMP != nil {
		if p, err := xmp.Parse(info.XMP); err == nil {
			t.applyXMP(p)
		}
	}

	if info.Width > 0 && info.Height > 0 {
		t.Width, t.Height = info.Width, info.Height
	}

	if mimetypes.IsHEIF(path) {
		t.Orientation = nil
		if info.HasRotation {
			o := map[int]int{0: 1, 90: 8, 180: 3, 270: 6}[info.Rotation]
			t.Orientation = &o
		}
	}

	probed := t.IsVideo || strings.EqualFold(mimetypes.Extension(path), ".gif")
	if probed && ffprobe != "" {
		v, err := video.Probe(ctx, ffprobe, path)
		switch {
		case err == nil:
			t.applyVideo(v)
		case errors.Is(err, video.ErrNoProbe):
		default:
			return nil, err
		}
	}
	if !probed && mimetypes.IsPossiblyAnimated(path) {
		if secs, ok := animationSeconds(f, st.Size(), info); ok {
			t.Duration = &secs
		}
	}

	t.inferZone()

	return t, nil
}

func (t *Tags) dimensionsFromTIFF(tf *exif.TIFF) {
	if w, okW := tf.Exif[0xa002].Uint(0); okW {
		if h, okH := tf.Exif[0xa003].Uint(0); okH && w > 0 && h > 0 {
			t.Width, t.Height = int(w), int(h)
			return
		}
	}

	best := 0
	for _, ifd := range append([]exif.IFD{tf.IFD0}, tf.SubIFDs...) {
		w, okW := ifd[0x0100].Uint(0)
		h, okH := ifd[0x0101].Uint(0)
		if okW && okH && int(w*h) > best {
			best = int(w * h)
			t.Width, t.Height = int(w), int(h)
		}
	}
}

func ReadSidecar(path string) (*Tags, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	t := newTags()
	p, err := xmp.Parse(data)
	if err != nil {
		return t, nil
	}
	t.applyXMP(p)
	t.inferZone()

	return t, nil
}

func (t *Tags) Merge(sidecar *Tags) {
	if sidecar == nil {
		return
	}

	if _, ok := sidecar.FirstDate(); ok {
		t.Dates = sidecar.Dates
		t.Zone, t.TZ, t.ZoneSource = sidecar.Zone, sidecar.TZ, sidecar.ZoneSource
	} else if sidecar.Zone != nil || sidecar.TZ != "" {
		t.Zone, t.TZ, t.ZoneSource = sidecar.Zone, sidecar.TZ, sidecar.ZoneSource
	}

	str := func(dst *string, v string) {
		if v != "" {
			*dst = v
		}
	}
	str(&t.Make, sidecar.Make)
	str(&t.Model, sidecar.Model)
	str(&t.LensModel, sidecar.LensModel)
	str(&t.Description, sidecar.Description)
	str(&t.ProjectionType, sidecar.ProjectionType)
	if sidecar.Orientation != nil {
		t.Orientation = sidecar.Orientation
	}
	if sidecar.Latitude != nil {
		t.Latitude, t.Longitude = sidecar.Latitude, sidecar.Longitude
	}
	if sidecar.Rating != nil {
		t.Rating = sidecar.Rating
	}
	if sidecar.Keywords != nil {
		t.Keywords = sidecar.Keywords
	}
}

func Sidewards(orientation *int) bool {
	return orientation != nil && slices.Contains([]int{5, 6, 7, 8}, *orientation)
}

func Stat(path string) (size int64, modified time.Time, born time.Time, err error) {
	st, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}, time.Time{}, err
	}

	return st.Size(), st.ModTime(), birthTime(path), nil
}
