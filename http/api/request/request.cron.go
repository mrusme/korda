package request

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	cronPresets = map[string]string{
		"@yearly": "0 0 0 1 1 *", "@monthly": "0 0 0 1 * *", "@weekly": "0 0 0 * * 0", "@daily": "0 0 0 * * *",
		"@hourly": "0 0 * * * *", "@minutely": "0 * * * * *", "@secondly": "* * * * * *",
		"@weekdays": "0 0 0 * * 1-5", "@weekends": "0 0 0 * * 0,6",
	}
	cronAliases = map[string]int{
		"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10,
		"nov": 11, "dec": 12, "sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
	}
	cronUnits    = []string{"second", "minute", "hour", "dayOfMonth", "month", "dayOfWeek"}
	cronBounds   = [][2]int{{0, 59}, {0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}
	cronDefaults = []string{"0", "*", "*", "*", "*", "*"}
	cronLetters  = regexp.MustCompile(`(?i)[a-z]{1,3}`)
	cronRange    = regexp.MustCompile(`^(\d+)(?:-(\d+))?(?:/(\d+))?$`)
	cronSpaces   = regexp.MustCompile(`\s+`)
)

func cronTime(source string) error {
	source = strings.ToLower(source)
	if preset, ok := cronPresets[source]; ok {
		source = preset
	}

	var aliasErr error
	source = cronLetters.ReplaceAllStringFunc(source, func(alias string) string {
		if v, ok := cronAliases[alias]; ok {
			return strconv.Itoa(v)
		}
		if aliasErr == nil {
			aliasErr = errors.New("Unknown alias: " + alias)
		}
		return alias
	})
	if aliasErr != nil {
		return aliasErr
	}

	units := cronSpaces.Split(jsTrim(source), -1)
	if len(units) < len(cronUnits)-1 {
		return errors.New("Too few fields")
	}
	if len(units) > len(cronUnits) {
		return errors.New("Too many fields")
	}

	for i, unit := range cronUnits {
		value := cronDefaults[i]
		if j := i - (len(cronUnits) - len(units)); j >= 0 {
			value = units[j]
		}
		if err := cronField(value, unit, cronBounds[i]); err != nil {
			return err
		}
	}

	return nil
}

func cronField(value string, unit string, bounds [2]int) error {
	low, high := bounds[0], bounds[1]
	for field := range strings.SplitSeq(value, ",") {
		if i := strings.Index(field, "*"); i > 0 {
			return errors.New("Field (" + field + ") has an invalid wildcard expression")
		}
	}

	value = strings.ReplaceAll(value, "*", strconv.Itoa(low)+"-"+strconv.Itoa(high))
	for part := range strings.SplitSeq(value, ",") {
		m := cronRange.FindStringSubmatch(part)
		if m == nil {
			return errors.New("Field (" + unit + ") cannot be parsed")
		}

		lower, _ := strconv.Atoi(m[1])
		if m[3] != "" {
			if step, _ := strconv.Atoi(m[3]); step == 0 {
				return errors.New("Field (" + unit + ") has a step of zero")
			}
		}
		if m[2] != "" {
			upper, _ := strconv.Atoi(m[2])
			if lower > upper {
				return errors.New("Field (" + unit + ") has an invalid range")
			}
			if lower < low || upper > high {
				return errors.New("Field value (" + value + ") is out of range")
			}
			continue
		}
		if lower < low || lower > high {
			return errors.New("Field value (" + value + ") is out of range")
		}
	}

	return nil
}

func cronExpression(raw any) string {
	s, _ := raw.(string)
	if err := cronTime(s); err != nil {
		return "Invalid cron expression. " + err.Error()
	}

	return ""
}
