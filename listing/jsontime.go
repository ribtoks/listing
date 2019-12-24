package main

import "time"

const jsonTimeLayout = "2006-01-02T15:04:05+07:00"

// JSONTime is the time.Time with JSON marshal and unmarshal capability
type JSONTime struct {
	time.Time
}

// UnmarshalJSON will unmarshal using 2006-01-02T15:04:05+07:00 layout
func (t *JSONTime) UnmarshalJSON(b []byte) error {
	parsed, err := time.Parse(jsonTimeLayout, string(b))
	if err != nil {
		return err
	}

	t.Time = parsed
	return nil
}

// MarshalJSON will marshal using 2006-01-02T15:04:05+07:00 layout
func (t *JSONTime) MarshalJSON() ([]byte, error) {
	s := t.Format(jsonTimeLayout)
	return []byte(s), nil
}
