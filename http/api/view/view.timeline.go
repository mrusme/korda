package view

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	logictimeline "tty.fail/mrus/korda/logic/root/timeline"
	"tty.fail/mrus/korda/models/asset"
	repoasset "tty.fail/mrus/korda/services/repositories/asset"
)

type TimeBucket struct {
	TimeBucket string `json:"timeBucket"`
	Count      int64  `json:"count"`
}

func NewTimeBuckets(list []repoasset.TimeBucket) []TimeBucket {
	out := make([]TimeBucket, 0, len(list))
	for _, b := range list {
		out = append(out, TimeBucket{TimeBucket: b.TimeBucket, Count: b.Count})
	}

	return out
}

func pgString(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if c < 0x20 {
				buf.WriteString(`\u00`)
				buf.WriteByte("0123456789abcdef"[c>>4])
				buf.WriteByte("0123456789abcdef"[c&0xf])
			} else {
				buf.WriteByte(c)
			}
		}
	}
	buf.WriteByte('"')

	return buf.String()
}

func pgTimestamp(t time.Time) string {
	return `"` + t.UTC().Truncate(time.Microsecond).Format("2006-01-02T15:04:05.999999") + `"`
}

func pgFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func pgRatio(width, height int64) string {
	if width <= 0 || height <= 0 {
		return "1"
	}
	thousandths := (2000*width + height) / (2 * height)

	return fmt.Sprintf("%d.%03d", thousandths/1000, thousandths%1000)
}

func offsetHours(a *asset.Asset) string {
	seconds := a.LocalDateTime.Truncate(time.Microsecond).Sub(a.FileCreatedAt.Truncate(time.Microsecond)).Seconds()
	return pgFloat(float64(float32(seconds)) / 3600)
}

func TimeBucketBody(b *logictimeline.Bucket) []byte {
	var buf bytes.Buffer
	first := true
	column := func(name string, value func(x *repoasset.TimelineAsset) string) {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		buf.WriteString(`"` + name + `":[`)
		for i := range b.Assets {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(value(&b.Assets[i]))
		}
		buf.WriteByte(']')
	}
	null := func(valid bool, value string) string {
		if !valid {
			return "null"
		}
		return value
	}

	buf.WriteByte('{')
	column("duration", func(x *repoasset.TimelineAsset) string {
		return null(x.Duration.Valid, strconv.FormatInt(x.Duration.V, 10))
	})
	column("id", func(x *repoasset.TimelineAsset) string { return `"` + x.ID.String() + `"` })
	column("visibility", func(x *repoasset.TimelineAsset) string { return pgString(string(x.Visibility)) })
	column("isFavorite", func(x *repoasset.TimelineAsset) string {
		return strconv.FormatBool(x.IsFavorite && x.OwnerID == b.Viewer)
	})
	column("isImage", func(x *repoasset.TimelineAsset) string { return strconv.FormatBool(x.Type == asset.TypeImage) })
	column("isTrashed", func(x *repoasset.TimelineAsset) string { return strconv.FormatBool(x.DeletedAt.Valid) })
	column("livePhotoVideoId", func(x *repoasset.TimelineAsset) string {
		return null(x.LivePhotoVideoID.Valid, `"`+x.LivePhotoVideoID.UUID.String()+`"`)
	})
	column("fileCreatedAt", func(x *repoasset.TimelineAsset) string { return pgTimestamp(x.FileCreatedAt) })
	column("localOffsetHours", func(x *repoasset.TimelineAsset) string { return offsetHours(&x.Asset) })
	column("createdAt", func(x *repoasset.TimelineAsset) string { return pgTimestamp(x.CreatedAt) })
	column("ownerId", func(x *repoasset.TimelineAsset) string { return `"` + x.OwnerID.String() + `"` })
	column("projectionType", func(x *repoasset.TimelineAsset) string {
		return null(x.ProjectionType.Valid, pgString(x.ProjectionType.V))
	})
	column("ratio", func(x *repoasset.TimelineAsset) string {
		if !x.Width.Valid || !x.Height.Valid {
			return "1"
		}
		return pgRatio(x.Width.V, x.Height.V)
	})
	column("status", func(x *repoasset.TimelineAsset) string { return pgString(string(x.Status)) })
	column("thumbhash", func(x *repoasset.TimelineAsset) string {
		return null(len(x.Thumbhash) > 0, `"`+base64.StdEncoding.EncodeToString(x.Thumbhash)+`"`)
	})
	if b.ShowExif {
		column("city", func(x *repoasset.TimelineAsset) string { return null(x.City.Valid, pgString(x.City.V)) })
		column("country", func(x *repoasset.TimelineAsset) string { return null(x.Country.Valid, pgString(x.Country.V)) })
	}
	if b.WithCoordinates {
		column("latitude", func(x *repoasset.TimelineAsset) string { return null(x.Latitude.Valid, pgFloat(x.Latitude.V)) })
		column("longitude", func(x *repoasset.TimelineAsset) string { return null(x.Longitude.Valid, pgFloat(x.Longitude.V)) })
	}
	if b.WithStacked {
		buf.WriteString(`,"stack":[`)
		for i := range b.Assets {
			x := &b.Assets[i]
			if i > 0 {
				buf.WriteString(", ")
			}
			if !x.StackID.Valid || x.StackCount == 0 {
				buf.WriteString("null")
				continue
			}
			if i > 0 {
				buf.WriteString("\n ")
			}
			buf.WriteString(`["` + x.StackID.UUID.String() + `","` + strconv.FormatInt(x.StackCount, 10) + `"]`)
		}
		buf.WriteByte(']')
	}
	buf.WriteByte('}')

	return buf.Bytes()
}
