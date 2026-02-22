package route

import "time"

type Route struct {
	ID          string    `json:"id"`
	ShipmentID  string    `json:"shipment_id"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	DistanceKm  float64   `json:"distance_km"`
	DurationH   float64   `json:"duration_h"`
	Cost        float64   `json:"cost"`
	CreatedAt   time.Time `json:"created_at"`
}
