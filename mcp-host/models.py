"""Execution plan models for MCP Planner."""

from __future__ import annotations

from datetime import datetime, timezone
from enum import Enum
from uuid import uuid4

from pydantic import BaseModel, Field


class StepStatus(str, Enum):
    """Status of a single plan step."""

    PENDING = "pending"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"
    SKIPPED = "skipped"


class PlanStatus(str, Enum):
    """Overall plan status."""

    RUNNING = "running"
    COMPLETED = "completed"
    PARTIAL_FAILURE = "partial_failure"
    FAILED = "failed"


class PlanStep(BaseModel):
    """A single step in an execution plan."""

    id: str = Field(default_factory=lambda: uuid4().hex[:12])
    tool: str
    params: dict = Field(default_factory=dict)
    result: dict | None = None
    status: StepStatus = StepStatus.PENDING
    started_at: datetime | None = None
    finished_at: datetime | None = None
    duration_ms: int | None = None
    error: str | None = None


class ExecutionPlan(BaseModel):
    """Execution plan that tracks tool calls for a single chat turn."""

    id: str = Field(default_factory=lambda: uuid4().hex)
    session_id: str
    intent: str
    steps: list[PlanStep] = Field(default_factory=list)
    status: PlanStatus = PlanStatus.RUNNING
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))
    finished_at: datetime | None = None

    def add_step(self, tool: str, params: dict) -> PlanStep:
        """Create and add a new pending step."""
        step = PlanStep(tool=tool, params=params)
        self.steps.append(step)
        return step

    def start_step(self, step: PlanStep) -> None:
        """Mark step as running."""
        step.status = StepStatus.RUNNING
        step.started_at = datetime.now(timezone.utc)

    def complete_step(self, step: PlanStep, result: dict) -> None:
        """Mark step as successfully completed."""
        step.status = StepStatus.SUCCESS
        step.result = result
        step.finished_at = datetime.now(timezone.utc)
        if step.started_at:
            step.duration_ms = int((step.finished_at - step.started_at).total_seconds() * 1000)

    def fail_step(self, step: PlanStep, error: str) -> None:
        """Mark step as failed."""
        step.status = StepStatus.FAILED
        step.error = error
        step.finished_at = datetime.now(timezone.utc)
        if step.started_at:
            step.duration_ms = int((step.finished_at - step.started_at).total_seconds() * 1000)

    def finalize(self) -> None:
        """Set overall plan status based on step outcomes."""
        self.finished_at = datetime.now(timezone.utc)
        statuses = {s.status for s in self.steps}
        if not self.steps:
            self.status = PlanStatus.COMPLETED
        elif statuses == {StepStatus.SUCCESS}:
            self.status = PlanStatus.COMPLETED
        elif statuses == {StepStatus.FAILED}:
            self.status = PlanStatus.FAILED
        elif StepStatus.FAILED in statuses:
            self.status = PlanStatus.PARTIAL_FAILURE
        else:
            self.status = PlanStatus.COMPLETED
