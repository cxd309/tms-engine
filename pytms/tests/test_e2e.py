import shutil

import pytest

from pytms import Network, RouteStop, Service, Vehicle

pytestmark = pytest.mark.e2e


@pytest.fixture
def two_node_network():
    net = Network()
    net.add_node("A")
    net.add_node("B")
    net.add_edge("A", "B", length=1000.0)
    net.add_edge("B", "A", length=1000.0)
    vehicle = Vehicle(name="Train", length=50.0, v_max=20.0, a_acc=0.5, a_dcc=0.7)
    svc = Service("S1", vehicle, "A", [RouteStop("B", 30.0), RouteStop("A", 30.0)])
    net.add_service(svc)
    return net


@pytest.mark.skipif(
    shutil.which("tms-engine") is None,
    reason="tms-engine binary not on PATH",
)
class TestE2E:
    def test_run_returns_result(self, two_node_network):
        result = two_node_network.run(run_time=300.0, time_step=1.0)
        assert result is not None

    def test_result_has_output(self, two_node_network):
        result = two_node_network.run(run_time=300.0, time_step=1.0)
        assert len(result.output) > 0

    def test_result_timesteps(self, two_node_network):
        result = two_node_network.run(run_time=300.0, time_step=1.0)
        timestamps = [row["timestamp"] for row in result.output]
        assert timestamps[0] == 0.0
        assert timestamps[-1] == pytest.approx(300.0)

    def test_service_present_in_output(self, two_node_network):
        result = two_node_network.run(run_time=300.0, time_step=1.0)
        first_row = result.output[0]
        service_ids = [s["service_id"] for s in first_row["service_logs"]]
        assert "S1" in service_ids
