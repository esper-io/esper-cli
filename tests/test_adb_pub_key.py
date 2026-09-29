"""
Unit tests for ADB public key resolution and loading logic.

Covers:
  - esper.ext.remoteadb_api._load_adb_pub_key  (P1: key absent → adb start-server)
  - esper.cli.state._resolve_adb_pub_key_path   (P2: ADB_VENDOR_KEYS support)
"""
import os
import subprocess
from unittest.mock import MagicMock, patch

import pytest

from esper.ext.remoteadb_api import RemoteADBError, _load_adb_pub_key
from esper.cli.state import _resolve_adb_pub_key_path


# ---------------------------------------------------------------------------
# _load_adb_pub_key
# ---------------------------------------------------------------------------

class TestLoadAdbPubKey:

    def test_reads_existing_key(self, tmp_path):
        key = tmp_path / "adbkey.pub"
        key.write_text("AAAAB3Nza host@example")
        assert _load_adb_pub_key(str(key), None) == "AAAAB3Nza host@example"

    def test_empty_path_falls_back_to_default(self, tmp_path):
        default = tmp_path / "adbkey.pub"
        default.write_text("DEFAULT_KEY")
        with patch("esper.ext.remoteadb_api._DEFAULT_ADB_PUB_KEY", str(default)):
            assert _load_adb_pub_key("", None) == "DEFAULT_KEY"

    def test_missing_key_adb_not_installed_returns_empty(self, tmp_path):
        # Key pre-authorization is optional; hosts without adb must still work.
        missing = str(tmp_path / "adbkey.pub")
        log = MagicMock()
        with patch("subprocess.run", side_effect=FileNotFoundError):
            result = _load_adb_pub_key(missing, log)
        assert result == ""
        log.warning.assert_called_once()

    def test_missing_key_adb_start_server_times_out_returns_empty(self, tmp_path):
        missing = str(tmp_path / "adbkey.pub")
        log = MagicMock()
        with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("adb", 30)):
            result = _load_adb_pub_key(missing, log)
        assert result == ""
        log.warning.assert_called_once()

    def test_missing_key_adb_runs_but_file_still_absent_returns_empty(self, tmp_path):
        missing = str(tmp_path / "adbkey.pub")
        log = MagicMock()
        with patch("subprocess.run", return_value=MagicMock(returncode=0)):
            result = _load_adb_pub_key(missing, log)
        assert result == ""
        log.warning.assert_called_once()

    def test_empty_key_file_raises(self, tmp_path):
        key = tmp_path / "adbkey.pub"
        key.write_bytes(b"")
        with pytest.raises(RemoteADBError, match="empty"):
            _load_adb_pub_key(str(key), None)

    def test_whitespace_only_key_file_raises(self, tmp_path):
        key = tmp_path / "adbkey.pub"
        key.write_text("   \n\t\n")
        with pytest.raises(RemoteADBError, match="empty"):
            _load_adb_pub_key(str(key), None)

    def test_missing_key_adb_generates_key(self, tmp_path):
        key = tmp_path / "adbkey.pub"

        def fake_start_server(*args, **kwargs):
            key.write_text("GENERATED_KEY host@example")
            return MagicMock(returncode=0)

        with patch("subprocess.run", side_effect=fake_start_server):
            assert _load_adb_pub_key(str(key), None) == "GENERATED_KEY host@example"

    def test_debug_logged_before_start_server(self, tmp_path):
        missing = str(tmp_path / "adbkey.pub")
        log = MagicMock()
        with patch("subprocess.run", side_effect=FileNotFoundError):
            result = _load_adb_pub_key(missing, log)
        assert result == ""
        log.debug.assert_called()

    def test_no_vendor_key_warning_when_adb_vendor_keys_unset(self, tmp_path):
        key = tmp_path / "adbkey.pub"
        key.write_text("KEY")
        log = MagicMock()
        env = {k: v for k, v in os.environ.items() if k != "ADB_VENDOR_KEYS"}
        with patch.dict(os.environ, env, clear=True):
            _load_adb_pub_key(str(key), log)
        log.warning.assert_not_called()

    def test_vendor_key_match_no_warning(self, tmp_path):
        vendor_priv = tmp_path / "vendor_key"
        vendor_pub = tmp_path / "vendor_key.pub"
        vendor_pub.write_text("VENDOR_KEY")
        log = MagicMock()
        with patch.dict(os.environ, {"ADB_VENDOR_KEYS": str(vendor_priv)}):
            _load_adb_pub_key(str(vendor_pub), log)
        log.warning.assert_not_called()

    def test_vendor_key_mismatch_logs_warning(self, tmp_path):
        default_key = tmp_path / "adbkey.pub"
        default_key.write_text("DEFAULT_KEY")
        vendor_priv = tmp_path / "other_key"
        log = MagicMock()
        with patch.dict(os.environ, {"ADB_VENDOR_KEYS": str(vendor_priv)}):
            _load_adb_pub_key(str(default_key), log)
        log.warning.assert_called_once()
        assert "ESPER_ADB_PUB_KEY" in log.warning.call_args[0][0]


# ---------------------------------------------------------------------------
# _resolve_adb_pub_key_path
# ---------------------------------------------------------------------------

class TestResolveAdbPubKeyPath:

    def test_esper_env_var_wins(self, tmp_path):
        override = str(tmp_path / "custom.pub")
        with patch.dict(os.environ, {"ESPER_ADB_PUB_KEY": override, "ADB_VENDOR_KEYS": "/some/key"}):
            assert _resolve_adb_pub_key_path() == override

    def test_vendor_key_used_when_pub_exists(self, tmp_path):
        vendor_priv = tmp_path / "vendor_key"
        vendor_pub = tmp_path / "vendor_key.pub"
        vendor_pub.write_text("VENDOR")
        env = {"ADB_VENDOR_KEYS": str(vendor_priv)}
        with patch.dict(os.environ, env):
            with patch.dict(os.environ, {"ESPER_ADB_PUB_KEY": ""}):
                result = _resolve_adb_pub_key_path()
        assert result == str(vendor_pub)

    def test_falls_back_when_vendor_pub_missing(self, tmp_path):
        vendor_priv = tmp_path / "vendor_key"  # no matching .pub
        env = {"ADB_VENDOR_KEYS": str(vendor_priv)}
        with patch.dict(os.environ, env):
            with patch.dict(os.environ, {"ESPER_ADB_PUB_KEY": ""}):
                result = _resolve_adb_pub_key_path()
        assert result == os.path.expanduser("~/.android/adbkey.pub")

    def test_falls_back_when_no_vendor_keys_set(self):
        env = {k: v for k, v in os.environ.items()
               if k not in ("ESPER_ADB_PUB_KEY", "ADB_VENDOR_KEYS")}
        with patch.dict(os.environ, env, clear=True):
            result = _resolve_adb_pub_key_path()
        assert result == os.path.expanduser("~/.android/adbkey.pub")

    def test_first_vendor_key_used_when_multiple(self, tmp_path):
        first_priv = tmp_path / "key1"
        first_pub = tmp_path / "key1.pub"
        first_pub.write_text("FIRST")
        second_priv = tmp_path / "key2"
        second_pub = tmp_path / "key2.pub"
        second_pub.write_text("SECOND")
        vendor_keys = os.pathsep.join([str(first_priv), str(second_priv)])
        with patch.dict(os.environ, {"ADB_VENDOR_KEYS": vendor_keys, "ESPER_ADB_PUB_KEY": ""}):
            result = _resolve_adb_pub_key_path()
        assert result == str(first_pub)

    def test_second_vendor_key_used_when_first_pub_missing(self, tmp_path):
        # /missing/key has no .pub; /valid/key.pub exists — should use the valid one.
        missing_priv = tmp_path / "missing_key"   # no matching .pub
        valid_priv = tmp_path / "valid_key"
        valid_pub = tmp_path / "valid_key.pub"
        valid_pub.write_text("VALID")
        vendor_keys = os.pathsep.join([str(missing_priv), str(valid_priv)])
        with patch.dict(os.environ, {"ADB_VENDOR_KEYS": vendor_keys, "ESPER_ADB_PUB_KEY": ""}):
            result = _resolve_adb_pub_key_path()
        assert result == str(valid_pub)
