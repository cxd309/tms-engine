# pytms

A lightweight, open-source transport simulator for research.

pytms lets you define a transport network and run multi-service simulations in Python. It is built around a Go simulation engine and a [NetworkX](https://networkx.org/)-based API for building networks.

---

## Features

- Mode-independent — model rail, bus, or any dedicated-corridor system with fixed stopping patterns
- Braking-distance separation between services
- Edge speed limits with automatic lookahead braking
- Staggered service departures
- Extensible kinematics (constant acceleration/deceleration today, more planned)
- NetworkX-compatible — use any nx tool for analysis, path-finding, or visualisation

---

## Installation

```bash
pip install pytms
```

Requires Python 3.12 or later.

---

## Quick start

```python
import pytms

# Build a network
net = pytms.Network()
net.add_node("A")
net.add_node("B")
net.add_edge("A", "B", length=1000.0)           # length in metres
net.add_edge("B", "A", length=1000.0, speed_limit=30)

# Define a vehicle
vehicle = pytms.Vehicle(
    name="Train",
    length=50.0,       # metres
    v_max=50.0,        # m/s
    a_acc=0.5,         # m/s²
    a_dcc=0.7,         # m/s²
)

# Define a service
service = pytms.Service(
    service_id="S1",
    vehicle=vehicle,
    initial_position="A",
    route=[
        pytms.RouteStop("B", t_dwell=30.0),
        pytms.RouteStop("A", t_dwell=30.0),
    ],
)
net.add_service(service)

# Run the simulation
result = net.run(run_time=600.0, time_step=1.0)

print(result)
# SimulationResult(simulation_id='...', steps=601)

# Access the raw output
for row in result.output:
    print(row["timestamp"], row["service_logs"])
```

---

## Network

`pytms.Network` extends `networkx.DiGraph`. All standard NetworkX operations work including visualisations.

### Edge attributes

| Attribute     | Type    | Required | Description                      |
| ------------- | ------- | -------- | -------------------------------- |
| `length`      | `float` | Yes      | Edge length in metres            |
| `speed_limit` | `float` | No       | Maximum speed on this edge (m/s) |

---

## Services

A `Service` defines a vehicle travelling a repeated route between stops.

```python
service = pytms.Service(
    service_id="S1",          # unique identifier
    vehicle=vehicle,
    initial_position="A",     # starting node
    route=[                   # list of stops in order
        pytms.RouteStop("B", t_dwell=30.0),
        pytms.RouteStop("A", t_dwell=30.0),
    ],
    departure_delay=60.0,     # seconds before departing (optional, default 0)
)
```

Multiple services can be added to a network and will interact through braking-distance separation:

```python
net.add_service(service_a)
net.add_service(service_b)
result = net.run(run_time=600.0, time_step=1.0)
```

---

## Results

`net.run()` returns a `SimulationResult`:

```python
result.meta      # simulation metadata dict
result.output    # list of timestep dicts, one per time step
result.to_dict() # full raw output as a Python dict
```

Each entry in `result.output` has the shape:

```python
{
    "timestamp": 42.0,
    "service_logs": [
        {
            "service_id": "S1",
            "state": "accelerating",   # stationary | accelerating | cruising | decelerating | dwelling
            "velocity": 12.3,          # m/s
            "current_position": {"edge": "A->B", "distance_along_edge": 450.0},
            "next_stop": "B",
            "remaining_dwell": 0.0,
        }
    ]
}
```

---

## Architecture

pytms is a thin Python wrapper around [tms-engine](https://github.com/cxd309/tms-engine), a simulation engine written in Go. The Python layer handles network construction and result parsing; the engine handles all simulation logic. Communication is via JSON over a subprocess call.

---

## Licence

MIT
