package users

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

type HeatmapType string

const (
	HeatmapUpload HeatmapType = "Upload"
	HeatmapTaken  HeatmapType = "Taken"
)

func (HeatmapType) Enum() []string {
	return []string{string(HeatmapUpload), string(HeatmapTaken)}
}

type HeatmapInput struct {
	From *string      `json:"from" v:"date"`
	To   *string      `json:"to" v:"date"`
	Type *HeatmapType `json:"type"`
}

func (in *HeatmapInput) Ordered() bool {
	return in.From == nil || in.To == nil || *in.From <= *in.To
}

type HeatmapDay struct {
	Date  string
	Count int64
}

type Heatmap struct {
	From   string
	To     string
	Series []HeatmapDay
	Total  int64
}

func CalendarHeatmap(ctx context.Context, userID uuid.UUID, in *HeatmapInput, today time.Time) (*Heatmap, error) {
	to := today.UTC().Truncate(24 * time.Hour)
	if in.To != nil {
		parsed, err := time.Parse(dateLayout, *in.To)
		if err != nil {
			return nil, err
		}
		to = parsed
	}

	from := to.AddDate(0, 0, -52*7+1)
	if in.From != nil {
		parsed, err := time.Parse(dateLayout, *in.From)
		if err != nil {
			return nil, err
		}
		from = parsed
	}

	heatmap := &Heatmap{From: from.Format(dateLayout), To: to.Format(dateLayout), Series: []HeatmapDay{}}
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		heatmap.Series = append(heatmap.Series, HeatmapDay{Date: day.Format(dateLayout)})
	}

	return heatmap, nil
}
