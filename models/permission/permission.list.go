package permission

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type List []Permission

func (l List) Value() (driver.Value, error) {
	if l == nil {
		l = List{}
	}

	encoded, err := json.Marshal([]Permission(l))
	if err != nil {
		return nil, err
	}

	return string(encoded), nil
}

func (l *List) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	case nil:
		*l = List{}
		return nil
	default:
		return fmt.Errorf("permission list from %T", src)
	}

	return json.Unmarshal(raw, (*[]Permission)(l))
}
