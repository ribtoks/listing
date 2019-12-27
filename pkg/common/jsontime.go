package common

import (
	"fmt"
	"strings"
	"time"
)

const jsonTimeLayout = time.RFC3339

// JSONTime is the time.Time with JSON marshal and unmarshal capability
type JSONTime time.Time

// JsonTimeNow() is an alias to time.Now() casted to JSONTime
func JsonTimeNow() JSONTime {
	return JSONTime(time.Now().UTC())
}

// UnmarshalJSON will unmarshal using 2006-01-02T15:04:05+07:00 layout
func (t *JSONTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	nt, err := time.Parse(jsonTimeLayout, s)
	if err != nil {
		return err
	}
	*t = JSONTime(nt)
	return nil
}

// Time returns builtin time.Time for current JSONTime
func (t JSONTime) Time() time.Time {
	return time.Time(t)
}

// MarshalJSON will marshal using 2006-01-02T15:04:05+07:00 layout
func (t *JSONTime) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

// String returns the time in the custom format
func (t JSONTime) String() string {
	ct := time.Time(t)
	return fmt.Sprintf("%q", ct.Format(jsonTimeLayout))
}
