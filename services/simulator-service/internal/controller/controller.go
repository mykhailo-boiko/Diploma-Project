package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/state"
)

type Controller struct {
	state *state.State
	log   *zap.Logger
}

func New(st *state.State, log *zap.Logger) *Controller {
	return &Controller{state: st, log: log}
}

func (c *Controller) Status(w http.ResponseWriter, _ *http.Request) {
	httpresponse.OK(w, c.state.Snapshot())
}

type startRequest struct {
	Scenario string   `json:"scenario"`
	Speed    *float64 `json:"speed,omitempty"`
}

func (c *Controller) Start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.Scenario != "" && !state.IsValidScenario(req.Scenario) {
		httpresponse.InvalidField(w, "scenario",
			"one of: "+strings.Join(state.AllowedScenarios(), ", "), req.Scenario,
			"Use one of the allowed scenarios.",
			"steady", "holiday_spike", "carrier_failure", "demand_surge", "idle")
		return
	}
	scenario := state.ParseScenario(req.Scenario)
	if req.Scenario == "" {
		scenario = c.state.Scenario()
	}
	var speed float64
	if req.Speed == nil {
		speed = c.state.Speed()
	} else {
		speed = *req.Speed
		if speed < state.MinSpeed || speed > state.MaxSpeed {
			httpresponse.InvalidField(w, "speed",
				fmt.Sprintf("number between %.1f and %.1f", state.MinSpeed, state.MaxSpeed), speed,
				"Speed must be within the allowed range. Omit the field to keep the current speed.",
				"1", "5", "25", "50")
			return
		}
	}
	c.state.Start(scenario, speed)
	c.log.Info("Simulator started", zap.String("scenario", string(scenario)), zap.Float64("speed", speed))
	httpresponse.OK(w, c.state.Snapshot())
}

func (c *Controller) Stop(w http.ResponseWriter, _ *http.Request) {
	c.state.Stop()
	c.log.Info("Simulator stopped")
	httpresponse.OK(w, c.state.Snapshot())
}

type speedRequest struct {
	Speed float64 `json:"speed"`
}

func (c *Controller) SetSpeed(w http.ResponseWriter, r *http.Request) {
	var req speedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	if req.Speed < state.MinSpeed || req.Speed > state.MaxSpeed {
		httpresponse.InvalidField(w, "speed",
			fmt.Sprintf("number between %.1f and %.1f", state.MinSpeed, state.MaxSpeed), req.Speed,
			"Speed must be within the allowed range.",
			"1", "5", "25", "50")
		return
	}
	c.state.SetSpeed(req.Speed)
	httpresponse.OK(w, c.state.Snapshot())
}

type scenarioRequest struct {
	Scenario string `json:"scenario"`
}

func (c *Controller) SetScenario(w http.ResponseWriter, r *http.Request) {
	var req scenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid body")
		return
	}
	if !state.IsValidScenario(req.Scenario) {
		httpresponse.InvalidField(w, "scenario",
			"one of: "+strings.Join(state.AllowedScenarios(), ", "), req.Scenario,
			"Use one of the allowed scenarios.",
			"steady", "holiday_spike", "carrier_failure", "demand_surge", "idle")
		return
	}
	scenario := state.ParseScenario(req.Scenario)
	c.state.SetScenario(scenario)
	httpresponse.OK(w, c.state.Snapshot())
}
