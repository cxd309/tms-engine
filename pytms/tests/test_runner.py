import json
from unittest.mock import MagicMock, patch

import pytest

from pytms.runner import _find_binary, _platform_binary_name, run_engine


class TestPlatformBinaryName:
    def test_linux_amd64(self):
        with (
            patch("platform.system", return_value="Linux"),
            patch("platform.machine", return_value="x86_64"),
        ):
            assert _platform_binary_name() == "tms-engine-linux-amd64"

    def test_linux_arm64(self):
        with (
            patch("platform.system", return_value="Linux"),
            patch("platform.machine", return_value="aarch64"),
        ):
            assert _platform_binary_name() == "tms-engine-linux-arm64"

    def test_darwin_arm64(self):
        with (
            patch("platform.system", return_value="Darwin"),
            patch("platform.machine", return_value="arm64"),
        ):
            assert _platform_binary_name() == "tms-engine-darwin-arm64"

    def test_windows_amd64(self):
        with (
            patch("platform.system", return_value="Windows"),
            patch("platform.machine", return_value="AMD64"),
        ):
            assert _platform_binary_name() == "tms-engine-windows-amd64.exe"


class TestFindBinary:
    def test_finds_bundled_binary(self, tmp_path):
        fake_binary = tmp_path / "tms-engine-linux-amd64"
        fake_binary.touch()
        with (
            patch(
                "pytms.runner._platform_binary_name",
                return_value="tms-engine-linux-amd64",
            ),
            patch("pytms.runner.resources.files") as mock_files,
        ):
            files_mock = mock_files.return_value
            mock_ref = files_mock.__truediv__.return_value.__truediv__.return_value
            mock_ref.__str__ = lambda s: str(fake_binary)
            result = _find_binary()
        assert result == fake_binary

    def test_falls_back_to_path(self, tmp_path):
        fake_binary = tmp_path / "tms-engine"
        fake_binary.touch()
        # Make bundled binary lookup fail, fall back to PATH
        with (
            patch("pytms.runner.resources.files", side_effect=FileNotFoundError),
            patch("shutil.which", return_value=str(fake_binary)),
        ):
            result = _find_binary()
        assert result == fake_binary

    def test_raises_when_not_found(self):
        with (
            patch("pytms.runner.resources.files", side_effect=FileNotFoundError),
            patch("shutil.which", return_value=None),
        ):
            with pytest.raises(FileNotFoundError, match="tms-engine binary not found"):
                _find_binary()


class TestRunEngine:
    def _mock_completed_process(self, stdout: str, returncode: int = 0):
        proc = MagicMock()
        proc.returncode = returncode
        proc.stdout = stdout
        proc.stderr = ""
        return proc

    def test_returns_parsed_json(self, tmp_path):
        fake_binary = tmp_path / "tms-engine"
        fake_binary.touch()
        output = {"simulation_meta": {}, "output": []}
        with (
            patch("pytms.runner._find_binary", return_value=fake_binary),
            patch(
                "subprocess.run",
                return_value=self._mock_completed_process(json.dumps(output)),
            ),
        ):
            result = run_engine("{}")
        assert result == output

    def test_raises_on_nonzero_exit(self, tmp_path):
        fake_binary = tmp_path / "tms-engine"
        fake_binary.touch()
        proc = self._mock_completed_process("", returncode=1)
        proc.stderr = "something went wrong"
        with (
            patch("pytms.runner._find_binary", return_value=fake_binary),
            patch("subprocess.run", return_value=proc),
        ):
            with pytest.raises(RuntimeError, match="exited with code 1"):
                run_engine("{}")
