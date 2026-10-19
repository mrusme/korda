package asset

type Visibility string

const (
	VisibilityArchive  Visibility = "archive"
	VisibilityTimeline Visibility = "timeline"
	VisibilityHidden   Visibility = "hidden"
	VisibilityLocked   Visibility = "locked"
)

func (Visibility) Enum() []string {
	return []string{
		string(VisibilityArchive),
		string(VisibilityTimeline),
		string(VisibilityHidden),
		string(VisibilityLocked),
	}
}
