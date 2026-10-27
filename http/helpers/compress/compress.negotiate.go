package compress

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

var (
	supported      = []string{"br", "gzip", "deflate", "identity"}
	preferred      = []string{"br", "gzip"}
	simpleEncoding = regexp.MustCompile(`^\s*([^\s;]+)\s*(?:;(.*))?$`)
	floatPrefix    = regexp.MustCompile(`^\s*[+-]?(?:Infinity|\d+\.?\d*(?:[eE][+-]?\d+)?|\.\d+(?:[eE][+-]?\d+)?)`)
)

type spec struct {
	encoding string
	q        float64
	s        int
	o        int
	i        int
}

func parseFloat(s string) float64 {
	m := floatPrefix.FindString(s)
	if m == "" {
		return math.NaN()
	}
	v, err := strconv.ParseFloat(strings.Replace(strings.TrimSpace(m), "Infinity", "Inf", 1), 64)
	if err != nil {
		return math.NaN()
	}

	return v
}

func truthy(v float64) bool {
	return v != 0 && !math.IsNaN(v)
}

func or(values ...float64) float64 {
	for _, v := range values[:len(values)-1] {
		if truthy(v) {
			return v
		}
	}

	return values[len(values)-1]
}

func parseEncoding(s string, i int) (spec, bool) {
	m := simpleEncoding.FindStringSubmatch(s)
	if m == nil {
		return spec{}, false
	}

	q := 1.0
	if m[2] != "" {
		for param := range strings.SplitSeq(m[2], ";") {
			if p := strings.Split(strings.TrimSpace(param), "="); p[0] == "q" {
				if len(p) > 1 {
					q = parseFloat(p[1])
				} else {
					q = math.NaN()
				}
				break
			}
		}
	}

	return spec{encoding: m[1], q: q, i: i}, true
}

func specify(encoding string, s spec, index int) (spec, bool) {
	matched := 0
	switch {
	case strings.EqualFold(s.encoding, encoding):
		matched = 1
	case s.encoding != "*":
		return spec{}, false
	}

	return spec{encoding: encoding, i: index, o: s.i, q: s.q, s: matched}, true
}

func parseAccept(header string) []spec {
	parts := strings.Split(header, ",")
	accepts := []spec{}
	identity := false
	minQuality := 1.0
	for i, part := range parts {
		s, ok := parseEncoding(strings.TrimSpace(part), i)
		if !ok {
			continue
		}
		accepts = append(accepts, s)
		if _, ok := specify("identity", s, 0); ok {
			identity = true
		}
		q := s.q
		if !truthy(q) {
			q = 1
		}
		minQuality = math.Min(minQuality, q)
	}
	if !identity {
		accepts = append(accepts, spec{encoding: "identity", q: minQuality, i: len(parts)})
	}

	return accepts
}

func priority(encoding string, accepts []spec, index int) spec {
	best := spec{o: -1}
	for _, a := range accepts {
		s, ok := specify(encoding, a, index)
		if ok && or(float64(best.s-s.s), best.q-s.q, float64(best.o-s.o)) < 0 {
			best = s
		}
	}

	return best
}

func compare(a spec, b spec) float64 {
	if a.q != b.q {
		return b.q - a.q
	}
	ap, bp := slices.Index(preferred, a.encoding), slices.Index(preferred, b.encoding)
	switch {
	case ap == -1 && bp == -1:
		return or(float64(b.s-a.s), float64(a.o-b.o), float64(a.i-b.i))
	case ap != -1 && bp != -1:
		return float64(ap - bp)
	case ap == -1:
		return 1
	}

	return -1
}

func Negotiate(header string) string {
	if header == "" {
		return "identity"
	}

	accepts := parseAccept(header)
	ranked := []spec{}
	for i, encoding := range supported {
		if p := priority(encoding, accepts, i); p.q > 0 {
			ranked = append(ranked, p)
		}
	}
	if len(ranked) == 0 {
		return ""
	}
	sort.SliceStable(ranked, func(i, j int) bool { return compare(ranked[i], ranked[j]) < 0 })

	return ranked[0].encoding
}
