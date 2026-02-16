class SimulationResult:
    """Wraps the raw JSON output from tms-engine.

    Provides dict-style access right now, probably
    in future will do something like pandas df.
    """

    def __init__(self, raw: dict):
        self._raw = raw

    @property
    def meta(self) -> dict:
        return self._raw["simulation_meta"]

    @property
    def output(self) -> list[dict]:
        return self._raw["output"]

    def to_dict(self) -> dict:
        return self._raw

    def __repr__(self) -> str:
        n_steps = len(self.output)
        sim_id = self.meta.get("simulation_id", "?")
        return f"SimulationResult(simulation_id={sim_id!r}, steps={n_steps})"
