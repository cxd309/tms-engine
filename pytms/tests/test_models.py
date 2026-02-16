import pytest

from pytms import RouteStop, Service, Vehicle


@pytest.fixture
def vehicle():
    return Vehicle(name="Train", length=50.0, v_max=20.0, a_acc=0.5, a_dcc=0.7)


@pytest.fixture
def service(vehicle):
    return Service(
        service_id="S1",
        vehicle=vehicle,
        initial_position="A",
        route=[RouteStop("B", t_dwell=30.0), RouteStop("A", t_dwell=30.0)],
    )


class TestVehicle:
    def test_to_engine_dict_structure(self, vehicle):
        d = vehicle._to_engine_dict()
        assert d["name"] == "Train"
        assert d["length"] == 50.0
        assert d["kinematics"]["model"] == "constant"
        assert d["kinematics"]["v_max"] == 20.0
        assert d["kinematics"]["a_acc"] == 0.5
        assert d["kinematics"]["a_dcc"] == 0.7


class TestRouteStop:
    def test_defaults_to_zero_dwell(self):
        stop = RouteStop("A")
        assert stop.t_dwell == 0.0

    def test_custom_dwell(self):
        stop = RouteStop("B", t_dwell=60.0)
        assert stop.node_id == "B"
        assert stop.t_dwell == 60.0


class TestService:
    def test_to_engine_dict_structure(self, service):
        d = service._to_engine_dict()
        assert d["service_id"] == "S1"
        assert d["initial_position"] == "A"
        assert len(d["route"]) == 2
        assert d["route"][0] == {"node_id": "B", "t_dwell": 30.0}
        assert d["route"][1] == {"node_id": "A", "t_dwell": 30.0}
        assert "vehicle" in d

    def test_departure_delay_omitted_when_zero(self, service):
        d = service._to_engine_dict()
        assert "departure_delay" not in d

    def test_departure_delay_included_when_set(self, vehicle):
        svc = Service("S2", vehicle, "A", [RouteStop("B")], departure_delay=60.0)
        d = svc._to_engine_dict()
        assert d["departure_delay"] == 60.0
