
from typing import Annotated, Any

from mcp.server.fastmcp import FastMCP
from pydantic import Field

from http_client import api_get, api_post
from types_mcp import SimulatorScenario

SimulatorSpeed = Annotated[
    float,
    Field(gt=0.0, le=100.0, description="time-acceleration multiplier (0 < speed <= 100)"),
]

def register(mcp: FastMCP) -> None:

    @mcp.tool(description="Return the current state of the live simulator (admin-only). Use this when the user asks: 'is the simulator running', 'what scenario is active', 'how fast is time accelerated', 'how many orders has the bot created today'. Returns counters and metadata: enabled, scenario (idle | steady | holiday_spike | carrier_failure | demand_surge), speed (multiplier), uptime_secs, counters dict.")
    async def simulator_status() -> dict[str, Any]:
        return await api_get("/api/v1/simulator/status")

    @mcp.tool(description="Start the live simulator (admin-only). Args: scenario: One of idle | steady | holiday_spike | carrier_failure | demand_surge. speed: Time-acceleration multiplier (0 < speed <= 100). Typical: 1, 5, 10, 25, 50.")
    async def simulator_start(
        scenario: SimulatorScenario = "steady",
        speed: SimulatorSpeed = 1.0,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/simulator/start", {
            "scenario": scenario, "speed": speed,
        })

    @mcp.tool(description="Stop the live simulator (admin-only). All actors pause; counters retained.")
    async def simulator_stop() -> dict[str, Any]:
        return await api_post("/api/v1/simulator/stop", {})

    @mcp.tool(description="Adjust the simulator's time multiplier without restart. Args: speed: New multiplier (0 < speed <= 100).")
    async def simulator_set_speed(speed: SimulatorSpeed) -> dict[str, Any]:
        return await api_post("/api/v1/simulator/speed", {"speed": speed})

    @mcp.tool(description="Switch the active scenario without restart. Args: scenario: One of idle | steady | holiday_spike | carrier_failure | demand_surge.")
    async def simulator_set_scenario(scenario: SimulatorScenario) -> dict[str, Any]:
        return await api_post("/api/v1/simulator/scenario", {"scenario": scenario})
