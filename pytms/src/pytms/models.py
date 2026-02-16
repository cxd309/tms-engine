from dataclasses import dataclass
from typing import Annotated


@dataclass
class Vehicle:
    name: str
    length: Annotated[float, "metres"]
    v_max: Annotated[float, "m/s"]
    a_acc: Annotated[float, "m/s²"]
    a_dcc: Annotated[float, "m/s²"]

    def _to_engine_dict(self) -> dict:
        return {
            "name": self.name,
            "length": self.length,
            "kinematics": {
                "model": "constant",
                "v_max": self.v_max,
                "a_acc": self.a_acc,
                "a_dcc": self.a_dcc,
            },
        }


@dataclass
class RouteStop:
    node_id: str
    t_dwell: Annotated[float, "seconds"] = 0.0


@dataclass
class Service:
    service_id: str
    vehicle: Vehicle
    initial_position: str
    route: list[RouteStop]
    departure_delay: Annotated[float, "seconds"] = 0.0

    def _to_engine_dict(self) -> dict:
        d = {
            "service_id": self.service_id,
            "vehicle": self.vehicle._to_engine_dict(),
            "initial_position": self.initial_position,
            "route": [
                {"node_id": stop.node_id, "t_dwell": stop.t_dwell}
                for stop in self.route
            ],
        }
        if self.departure_delay:
            d["departure_delay"] = self.departure_delay
        return d
