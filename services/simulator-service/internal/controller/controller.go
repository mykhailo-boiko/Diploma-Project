package controller

import (
	"encoding/json"
	"net/http"

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
	Scenario string  `json:"scenario"`
	Speed    float64 `json:"speed"`
}

func (c *Controller) Start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	scenario := state.ParseScenario(req.Scenario)
	if req.Scenario == "" {
		scenario = c.state.Scenario()
	}
	speed := req.Speed
	if speed <= 0 {
		speed = c.state.Speed()
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
	if req.Speed <= 0 || req.Speed > 100 {
		httpresponse.BadRequest(w, "validation_error", "speed must be between 0 and 100")
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
	scenario := state.ParseScenario(req.Scenario)
	c.state.SetScenario(scenario)
	httpresponse.OK(w, c.state.Snapshot())
}
