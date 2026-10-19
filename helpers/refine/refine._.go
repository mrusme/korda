package refine

type Issue struct {
	Path    []any
	Message string
}

type Fields interface {
	Has(name string) bool
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
