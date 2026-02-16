import json
import uuid

import networkx as nx

from .models import Service
from .results import SimulationResult
from .runner import run_engine


class Network(nx.DiGraph):
    """A transport network. Extends nx.DiGraph with simulation capabilities.

    Nodes and edges are standard networkx — use any nx tools for analysis,
    path-finding, or visualisation. Edge attributes:
      - length (float, metres): required
      - speed_limit (float, m/s): optional

    Services are attached separately and run against the network.
    """

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self._services: list[Service] = []

    def add_service(self, service: Service) -> None:
        self._services.append(service)

    def run(
        self,
        run_time: float,
        time_step: float = 1.0,
        simulation_id: str | None = None,
    ) -> SimulationResult:
        """Run the simulation and return the result.

        Args:
            run_time: Total simulation duration in seconds.
            time_step: Timestep size in seconds.
            simulation_id: Optional identifier for the run; auto-generated if omitted.
        """
        sim_id = simulation_id or str(uuid.uuid4())
        payload = self._build_engine_input(sim_id, run_time, time_step)
        raw = run_engine(json.dumps(payload))
        return SimulationResult(raw)

    def _build_engine_input(
        self, simulation_id: str, run_time: float, time_step: float
    ) -> dict:
        nodes = [{"node_id": n} for n in self.nodes]

        edges = []
        for u, v, data in self.edges(data=True):
            if "length" not in data:
                raise ValueError(
                    f"Edge ({u!r}, {v!r}) is missing required attribute 'length'"
                )
            edge: dict = {
                "edge_id": f"{u}->{v}",
                "u": u,
                "v": v,
                "length": data["length"],
            }
            if "speed_limit" in data:
                edge["speed_limit"] = data["speed_limit"]
            edges.append(edge)

        return {
            "simulation_meta": {
                "simulation_id": simulation_id,
                "run_time": run_time,
                "time_step": time_step,
            },
            "graph_data": {
                "nodes": nodes,
                "edges": edges,
            },
            "service_list": [svc._to_engine_dict() for svc in self._services],
        }
