package config

import "testing"

type poolConfigStub map[string]any

func (s poolConfigStub) Env(key string, fallback ...any) any {
	if value, ok := s[key]; ok {
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}

func TestPoolLimitValidatesEnvironmentAndBounds(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		fallback int
		maximum  int
		want     int
	}{
		{name: "unset uses fallback", fallback: 32, maximum: 96, want: 32},
		{name: "valid override", value: "48", fallback: 32, maximum: 96, want: 48},
		{name: "invalid uses fallback", value: "many", fallback: 32, maximum: 96, want: 32},
		{name: "zero uses fallback", value: "0", fallback: 32, maximum: 96, want: 32},
		{name: "negative uses fallback", value: "-1", fallback: 32, maximum: 96, want: 32},
		{name: "too large is capped", value: "200", fallback: 32, maximum: 96, want: 96},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stub := poolConfigStub{}
			if test.value != nil {
				stub["DB_POOL"] = test.value
			}
			if got := poolLimit(stub, "DB_POOL", test.fallback, test.maximum); got != test.want {
				t.Fatalf("poolLimit() = %d, want %d", got, test.want)
			}
		})
	}
}
