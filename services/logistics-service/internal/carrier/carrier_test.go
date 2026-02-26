package carrier

import "testing"

func TestValidType(t *testing.T) {
	tests := []struct {
		name string
		t    Type
		want bool
	}{
		{name: "ground", t: TypeGround, want: true},
		{name: "air", t: TypeAir, want: true},
		{name: "sea", t: TypeSea, want: true},
		{name: "invalid", t: Type("bicycle"), want: false},
		{name: "empty", t: Type(""), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidType(tt.t)
			if got != tt.want {
				t.Errorf("ValidType(%s) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}
