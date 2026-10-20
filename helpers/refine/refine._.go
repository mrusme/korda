package refine

type Issue struct {
	Path         []any
	Message      string
	MessageFirst bool
}

type Fields interface {
	Has(name string) bool
	Get(name string) (any, bool)
}

type Refiner interface {
	Refine(in Fields) []Issue
}

type Piper interface {
	Pipe(in Fields) []Issue
}

type Strict struct{}

func At(path []any, message string) Issue {
	return Issue{Path: path, Message: message}
}

func Root(message string) Issue {
	return Issue{Path: []any{}, Message: message}
}

func Added(message string) Issue {
	return Issue{Path: []any{}, Message: message, MessageFirst: true}
}
