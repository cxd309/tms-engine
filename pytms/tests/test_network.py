import pytest

from pytms import Network, RouteStop, Service, Vehicle


@pytest.fixture
def vehicle():
    return Vehicle(name="Train", length=50.0, v_max=20.0, a_acc=0.5, a_dcc=0.7)


@pytest.fixture
def simple_network(vehicle):
    net = Network()
    net.add_node("A")
    net.add_node("B")
    net.add_edge("A", "B", length=1000.0)
    net.add_edge("B", "A", length=1000.0)
    svc = Service("S1", vehicle, "A", [RouteStop("B", 30.0), RouteStop("A", 30.0)])
    net.add_service(svc)
    return net


class TestNetworkBuild:
    def test_is_digraph(self, simple_network):
        import networkx as nx

        assert isinstance(simple_network, nx.DiGraph)

    def test_nodes_present(self, simple_network):
        assert "A" in simple_network.nodes
        assert "B" in simple_network.nodes

    def test_edge_attributes(self, simple_network):
        assert simple_network.edges["A", "B"]["length"] == 1000.0

    def test_speed_limit_optional(self):
        net = Network()
        net.add_node("A")
        net.add_node("B")
        net.add_edge("A", "B", length=500.0, speed_limit=10.0)
        assert net.edges["A", "B"]["speed_limit"] == 10.0


class TestBuildEngineInput:
    def test_meta_fields(self, simple_network):
        payload = simple_network._build_engine_input("test-run", 300.0, 1.0)
        meta = payload["simulation_meta"]
        assert meta["simulation_id"] == "test-run"
        assert meta["run_time"] == 300.0
        assert meta["time_step"] == 1.0

    def test_graph_nodes(self, simple_network):
        payload = simple_network._build_engine_input("x", 300.0, 1.0)
        node_ids = {n["node_id"] for n in payload["graph_data"]["nodes"]}
        assert node_ids == {"A", "B"}

    def test_graph_edges(self, simple_network):
        payload = simple_network._build_engine_input("x", 300.0, 1.0)
        edges = {e["edge_id"]: e for e in payload["graph_data"]["edges"]}
        assert "A->B" in edges
        assert edges["A->B"]["length"] == 1000.0
        assert "speed_limit" not in edges["A->B"]

    def test_speed_limit_included_when_set(self):
        net = Network()
        net.add_node("A")
        net.add_node("B")
        net.add_edge("A", "B", length=500.0, speed_limit=10.0)
        payload = net._build_engine_input("x", 300.0, 1.0)
        edges = {e["edge_id"]: e for e in payload["graph_data"]["edges"]}
        assert edges["A->B"]["speed_limit"] == 10.0

    def test_missing_length_raises(self):
        net = Network()
        net.add_node("A")
        net.add_node("B")
        net.add_edge("A", "B")  # no length
        with pytest.raises(ValueError, match="missing required attribute 'length'"):
            net._build_engine_input("x", 300.0, 1.0)

    def test_service_list(self, simple_network):
        payload = simple_network._build_engine_input("x", 300.0, 1.0)
        assert len(payload["service_list"]) == 1
        assert payload["service_list"][0]["service_id"] == "S1"
