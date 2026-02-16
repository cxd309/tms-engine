# tms-engine

The simulation engine powering [pytms](https://github.com/cxd309/pytms) — a lightweight, open-source transport simulator for research.

tms-engine is written in Go and compiled to two targets:

| Target      | Output            | Use case                          |
| ----------- | ----------------- | --------------------------------- |
| CLI binary  | `dist/tms-engine` | Python package (pytms), scripting |
| WebAssembly | `dist/sim.wasm`   | Browser-based tools               |

Both share the same JSON-in / JSON-out interface.

---

## How it works

The engine runs a fixed-timestep simulation loop with two passes per step:

1. **Safety pass** — every service computes its minimal Movement Authority (MA): the track ahead it physically needs to stop from its current velocity.
2. **Motion pass** — every service proposes its desired movement, has that proposal trimmed by the MA record and any edge speed limits, then updates its position, velocity, and state.

Services are separated by braking distance. A service cannot enter another service's safety envelope.

---

## Building

Requires Go 1.25+.

```bash
# CLI binary
make cli

# WebAssembly
make wasm

# Run Go tests
make test
```

Built artefacts are placed in `dist/`.

---

## JSON interface

Both targets accept a `SimulationInput` JSON object and return a `SimulationLog` JSON object.

### Input

```json
{
  "simulation_meta": {
    "simulation_id": "my-run",
    "run_time": 600.0,
    "time_step": 1.0
  },
  "graph_data": {
    "nodes": [
      { "node_id": "A" },
      { "node_id": "B" }
    ],
    "edges": [
      { "edge_id": "A->B", "u": "A", "v": "B", "length": 1000.0 },
      { "edge_id": "B->A", "u": "B", "v": "A", "length": 1000.0, "speed_limit": 10.0 }
    ]
  },
  "service_list": [
    {
      "service_id": "S1",
      "initial_position": "A",
      "departure_delay": 0.0,
      "route": [
        { "node_id": "B", "t_dwell": 30.0 },
        { "node_id": "A", "t_dwell": 30.0 }
      ],
      "vehicle": {
        "name": "Train",
        "length": 50.0,
        "kinematics": {
          "model": "constant",
          "v_max": 20.0,
          "a_acc": 0.5,
          "a_dcc": 0.7
        }
      }
    }
  ]
}
```

#### Field reference

**`simulation_meta`**

| Field           | Type   | Description                         |
| --------------- | ------ | ----------------------------------- |
| `simulation_id` | string | Identifier for the run              |
| `run_time`      | float  | Total simulation duration (seconds) |
| `time_step`     | float  | Timestep size (seconds)             |

**`graph_data.edges`**

| Field         | Type   | Required | Description                                               |
| ------------- | ------ | -------- | --------------------------------------------------------- |
| `edge_id`     | string | Yes      | Unique edge identifier                                    |
| `u`           | string | Yes      | Origin node ID                                            |
| `v`           | string | Yes      | Destination node ID                                       |
| `length`      | float  | Yes      | Edge length (metres)                                      |
| `speed_limit` | float  | No       | Maximum speed on this edge (m/s); omit for no restriction |

**`vehicle.kinematics`**

| Field   | Type   | Description                                   |
| ------- | ------ | --------------------------------------------- |
| `model` | string | `"constant"` (only supported model currently) |
| `v_max` | float  | Maximum speed (m/s)                           |
| `a_acc` | float  | Acceleration (m/s²)                           |
| `a_dcc` | float  | Deceleration (m/s², positive)                 |

**`service`**

| Field              | Type   | Required | Description                                             |
| ------------------ | ------ | -------- | ------------------------------------------------------- |
| `service_id`       | string | Yes      | Unique service identifier                               |
| `initial_position` | string | Yes      | Starting node ID                                        |
| `route`            | array  | Yes      | Ordered list of `{node_id, t_dwell}` stops              |
| `departure_delay`  | float  | No       | Seconds to hold stationary before departing (default 0) |

### Output

```json
{
  "simulation_meta": { ... },
  "output": [
    {
      "timestamp": 0.0,
      "service_logs": [
        {
          "service_id": "S1",
          "current_position": {"edge": "A->B", "distance_along_edge": 0.0},
          "state": "stationary",
          "velocity": 0.0,
          "remaining_dwell": 0.0,
          "next_stop": "B"
        }
      ]
    }
  ]
}
```

Service states: `stationary` | `accelerating` | `cruising` | `decelerating` | `dwelling`

---

## CLI usage

```bash
# From a file
./dist/tms-engine input.json

# From stdin
cat input.json | ./dist/tms-engine

# Pipe to jq for readable output
cat input.json | ./dist/tms-engine | jq .
```

---

## Architecture

```
tms-engine/
  internal/
    graph/        ← directed graph, Floyd-Warshall algorithm
    kinematics/   ← Vehicle motion model
    service/      ← Vehicle, Service, SimService state machine
    engine/       ← simulation loop, Movement Authority logic
  cmd/
    cli/          ← CLI binary entry point
    wasm/         ← WebAssembly entry point
  pytms/          ← Python package source (pytms)
```

Adding a new kinematics model requires only implementing the `kinematics.MotionModel` interface and registering it in `service.go` — the engine itself does not need to change.

---

## Related

- [pytms](https://github.com/cxd309/pytms) — Python package built on tms-engine

---

## Licence

MIT
