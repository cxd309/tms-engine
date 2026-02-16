import json
import platform
import shutil
import subprocess
from importlib import resources
from pathlib import Path


def _platform_binary_name() -> str:
    """Return the binary filename for the current platform."""
    system = platform.system().lower()
    machine = platform.machine().lower()

    arch = "arm64" if machine in ("arm64", "aarch64") else "amd64"

    if system == "windows":
        return f"tms-engine-windows-{arch}.exe"
    elif system == "darwin":
        return f"tms-engine-darwin-{arch}"
    else:
        return f"tms-engine-linux-{arch}"


def _find_binary() -> Path:
    """Locate the tms-engine binary.

    Search order:
    1. Bundled binary inside the installed package (production).
    2. 'tms-engine' on PATH (development: make cli).
    """
    # 1. Bundled binary (present in installed wheels).
    try:
        bin_ref = resources.files("pytms") / "bin" / _platform_binary_name()
        p = Path(str(bin_ref))
        if p.is_file():
            return p
    except (TypeError, FileNotFoundError, NotADirectoryError, ModuleNotFoundError):
        pass

    # 2. PATH fallback for development.
    on_path = shutil.which("tms-engine")
    if on_path:
        return Path(on_path)

    raise FileNotFoundError(
        "tms-engine binary not found. "
        "For development, build and add to PATH:\n"
        "  make cli && export PATH=$PATH:$(pwd)/dist\n"
        "Or install pytms with the bundled binary wheel."
    )


def run_engine(json_input: str) -> dict:
    """Pass json_input to the tms-engine binary and return the parsed JSON output."""
    binary = _find_binary()
    binary.chmod(binary.stat().st_mode | 0o111)  # ensure executable bit is set
    result = subprocess.run(
        [str(binary)],
        input=json_input,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"tms-engine exited with code {result.returncode}:\n{result.stderr.strip()}"
        )
    return json.loads(result.stdout)
