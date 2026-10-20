package query

import (
	"strconv"
	"strings"
)

const ChunkSize = 1000

func Placeholders[T any](args *[]any, values []T) string {
	marks := make([]string, 0, len(values))
	for _, v := range values {
		*args = append(*args, v)
		marks = append(marks, "$"+strconv.Itoa(len(*args)))
	}

	return strings.Join(marks, ", ")
}
