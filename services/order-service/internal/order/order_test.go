package order

import "testing"

func TestCanTransition_Valid(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
	}{
		{name: "pending to confirmed", from: StatusPending, to: StatusConfirmed},
		{name: "pending to cancelled", from: StatusPending, to: StatusCancelled},
		{name: "confirmed to processing", from: StatusConfirmed, to: StatusProcessing},
		{name: "confirmed to cancelled", from: StatusConfirmed, to: StatusCancelled},
		{name: "processing to shipped", from: StatusProcessing, to: StatusShipped},
		{name: "shipped to delivered", from: StatusShipped, to: StatusDelivered},
		{name: "shipped to returned", from: StatusShipped, to: StatusReturned},
		{name: "shipped to cancelled", from: StatusShipped, to: StatusCancelled},
		{name: "delivered to completed", from: StatusDelivered, to: StatusCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !CanTransition(tt.from, tt.to) {
				t.Errorf("expected transition %s -> %s to be valid", tt.from, tt.to)
			}
		})
	}
}

func TestCanTransition_Invalid(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
	}{
		{name: "pending to delivered", from: StatusPending, to: StatusDelivered},
		{name: "pending to shipped", from: StatusPending, to: StatusShipped},
		{name: "confirmed to delivered", from: StatusConfirmed, to: StatusDelivered},
		{name: "completed to pending", from: StatusCompleted, to: StatusPending},
		{name: "cancelled to pending", from: StatusCancelled, to: StatusPending},
		{name: "processing to confirmed", from: StatusProcessing, to: StatusConfirmed},
		{name: "pending to returned", from: StatusPending, to: StatusReturned},
		{name: "returned to pending", from: StatusReturned, to: StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if CanTransition(tt.from, tt.to) {
				t.Errorf("expected transition %s -> %s to be invalid", tt.from, tt.to)
			}
		})
	}
}
