package duration

import (
	"time"
	"encoding/json"
	"fmt"
)

// Duration duration for config parse
type Duration time.Duration

func (d Duration) String() string {
	dd := time.Duration(d)
	return dd.String()
}

// GoString  duration go string
func (d Duration) GoString() string {
	dd := time.Duration(d)
	return dd.String()
}

// Duration duration
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// UnmarshalText 字符串解析时间
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	dd, err := time.ParseDuration(string(text))
	if err == nil {
		*d = Duration(dd)
	}
	return err
}

func (d Duration) MarshalText() ([]byte, error) {
	dd := time.Duration(d)
	return []byte(dd.String()), nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%s", time.Duration(d)))
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return fmt.Errorf("invalid duration")
	}
}