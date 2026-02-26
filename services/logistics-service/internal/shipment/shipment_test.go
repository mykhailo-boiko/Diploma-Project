package shipment

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		{name: "created to picked_up", from: StatusCreated, to: StatusPickedUp, want: true},
		{name: "picked_up to in_transit", from: StatusPickedUp, to: StatusInTransit, want: true},
		{name: "in_transit to delivered", from: StatusInTransit, to: StatusDelivered, want: true},
		{name: "in_transit to failed", from: StatusInTransit, to: StatusFailed, want: true},
		{name: "delivered to returned", from: StatusDelivered, to: StatusReturned, want: true},
		{name: "failed to returned", from: StatusFailed, to: StatusReturned, want: true},
		{name: "created to delivered invalid", from: StatusCreated, to: StatusDelivered, want: false},
		{name: "created to in_transit invalid", from: StatusCreated, to: StatusInTransit, want: false},
		{name: "delivered to in_transit invalid", from: StatusDelivered, to: StatusInTransit, want: false},
		{name: "returned terminal", from: StatusReturned, to: StatusCreated, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanTransition(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
