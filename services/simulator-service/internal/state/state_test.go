package state

import "testing"

func TestParseScenario_KnownValues(t *testing.T) {
	cases := []struct {
		in   string
		want Scenario
	}{
		{"idle", ScenarioIdle},
		{"steady", ScenarioSteady},
		{"holiday_spike", ScenarioHolidaySpike},
		{"carrier_failure", ScenarioCarrierFailure},
		{"demand_surge", ScenarioDemandSurge},
	}
	for _, c := range cases {
		if got := ParseScenario(c.in); got != c.want {
			t.Errorf("ParseScenario(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseScenario_FallbackToSteady(t *testing.T) {
	if ParseScenario("nonsense") != ScenarioSteady {
		t.Errorf("expected default steady")
	}
	if ParseScenario("") != ScenarioSteady {
		t.Errorf("expected default steady for empty")
	}
}

func TestState_StartEnablesAndRecordsTime(t *testing.T) {
	s := New(ScenarioSteady, 1.0)
	if s.Enabled() {
		t.Fatalf("expected disabled by default")
	}
	s.Start(ScenarioHolidaySpike, 5.0)
	if !s.Enabled() {
		t.Errorf("expected enabled after Start")
	}
	if s.Scenario() != ScenarioHolidaySpike {
		t.Errorf("scenario = %q", s.Scenario())
	}
	if s.Speed() != 5.0 {
		t.Errorf("speed = %v", s.Speed())
	}
	if s.StartedAt().IsZero() {
		t.Errorf("started_at must be set")
	}
}

func TestState_SetSpeedIgnoresZeroAndNegative(t *testing.T) {
	s := New(ScenarioSteady, 1.0)
	s.SetSpeed(0)
	if s.Speed() != 1.0 {
		t.Errorf("zero speed should be rejected, got %v", s.Speed())
	}
	s.SetSpeed(-3)
	if s.Speed() != 1.0 {
		t.Errorf("negative speed should be rejected, got %v", s.Speed())
	}
	s.SetSpeed(10)
	if s.Speed() != 10 {
		t.Errorf("speed = %v, want 10", s.Speed())
	}
}

func TestState_SnapshotIncludesCounters(t *testing.T) {
	s := New(ScenarioSteady, 1.0)
	s.Start(ScenarioSteady, 2.0)
	s.Counters.OrdersCreated.Add(7)
	s.Counters.ShipmentsAdvanced.Add(3)
	snap := s.Snapshot()
	if snap.Counters.OrdersCreated != 7 || snap.Counters.ShipmentsAdvanced != 3 {
		t.Errorf("counters not propagated: %+v", snap.Counters)
	}
	if !snap.Enabled {
		t.Errorf("snapshot.Enabled should be true")
	}
}

func TestState_StopDisables(t *testing.T) {
	s := New(ScenarioSteady, 1.0)
	s.Start(ScenarioSteady, 1.0)
	s.Stop()
	if s.Enabled() {
		t.Errorf("Stop did not disable")
	}
}
