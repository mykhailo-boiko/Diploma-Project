
from models import ExecutionPlan, PlanStatus, PlanStep, StepStatus

class TestPlanStep:
    def test_defaults(self):
        step = PlanStep(tool="orders_list", params={"status": "pending"})
        assert step.tool == "orders_list"
        assert step.params == {"status": "pending"}
        assert step.status == StepStatus.PENDING
        assert step.result is None
        assert step.started_at is None
        assert step.finished_at is None
        assert step.duration_ms is None
        assert step.error is None
        assert len(step.id) == 12

    def test_unique_ids(self):
        s1 = PlanStep(tool="a", params={})
        s2 = PlanStep(tool="b", params={})
        assert s1.id != s2.id

class TestExecutionPlan:
    def test_create_plan(self):
        plan = ExecutionPlan(session_id="user1", intent="list orders")
        assert plan.session_id == "user1"
        assert plan.intent == "list orders"
        assert plan.status == PlanStatus.RUNNING
        assert plan.steps == []
        assert plan.finished_at is None
        assert len(plan.id) == 32

    def test_add_step(self):
        plan = ExecutionPlan(session_id="u1", intent="test")
        step = plan.add_step(tool="orders_list", params={"limit": 10})
        assert len(plan.steps) == 1
        assert step.tool == "orders_list"
        assert step.status == StepStatus.PENDING

    def test_start_step(self):
        plan = ExecutionPlan(session_id="u1", intent="test")
        step = plan.add_step(tool="t", params={})
        plan.start_step(step)
        assert step.status == StepStatus.RUNNING
        assert step.started_at is not None

    def test_complete_step(self):
        plan = ExecutionPlan(session_id="u1", intent="test")
        step = plan.add_step(tool="t", params={})
        plan.start_step(step)
        plan.complete_step(step, {"data": "ok"})
        assert step.status == StepStatus.SUCCESS
        assert step.result == {"data": "ok"}
        assert step.finished_at is not None
        assert step.duration_ms is not None
        assert step.duration_ms >= 0

    def test_fail_step(self):
        plan = ExecutionPlan(session_id="u1", intent="test")
        step = plan.add_step(tool="t", params={})
        plan.start_step(step)
        plan.fail_step(step, "connection refused")
        assert step.status == StepStatus.FAILED
        assert step.error == "connection refused"
        assert step.finished_at is not None
        assert step.duration_ms is not None

    def test_finalize_all_success(self):
        plan = ExecutionPlan(session_id="u1", intent="test")
        s1 = plan.add_step(tool="a", params={})
        s2 = plan.add_step(tool="b", params={})
        plan.start_step(s1)
        plan.complete_step(s1, {})
        plan.start_step(s2)
        plan.complete_step(s2, {})
        plan.finalize()
        assert plan.status == PlanStatus.COMPLETED
        assert plan.finished_at is not None

    def test_finalize_all_failed(self):
        plan = ExecutionPlan(session_id="u1", intent="test")
        s1 = plan.add_step(tool="a", params={})
        plan.start_step(s1)
        plan.fail_step(s1, "err")
        plan.finalize()
        assert plan.status == PlanStatus.FAILED

    def test_finalize_partial_failure(self):
        plan = ExecutionPlan(session_id="u1", intent="test")
        s1 = plan.add_step(tool="a", params={})
        s2 = plan.add_step(tool="b", params={})
        plan.start_step(s1)
        plan.complete_step(s1, {})
        plan.start_step(s2)
        plan.fail_step(s2, "err")
        plan.finalize()
        assert plan.status == PlanStatus.PARTIAL_FAILURE

    def test_finalize_empty_plan(self):
        plan = ExecutionPlan(session_id="u1", intent="test")
        plan.finalize()
        assert plan.status == PlanStatus.COMPLETED

    def test_serialization_roundtrip(self):
        plan = ExecutionPlan(session_id="u1", intent="test intent")
        step = plan.add_step(tool="orders_list", params={"status": "pending"})
        plan.start_step(step)
        plan.complete_step(step, {"orders": []})
        plan.finalize()

        json_str = plan.model_dump_json()
        restored = ExecutionPlan.model_validate_json(json_str)
        assert restored.id == plan.id
        assert restored.session_id == "u1"
        assert restored.intent == "test intent"
        assert len(restored.steps) == 1
        assert restored.steps[0].tool == "orders_list"
        assert restored.steps[0].status == StepStatus.SUCCESS
        assert restored.status == PlanStatus.COMPLETED
