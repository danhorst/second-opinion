import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class ServiceTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.home = Path(self.tmp.name)
        (self.home / "libexec").mkdir()
        (self.home / "bin").mkdir()
        self.launcher = self.home / "libexec/service"
        shutil.copy(ROOT / "scripts/second-opinion-mcp-service", self.launcher)
        binary = self.home / "bin/second-opinion"
        binary.write_text("#!/bin/bash\nprintf '%s\\n' \"$SECOND_OPINION_MCP_TOKEN\" \"$PATH\" \"$@\"\nexit 17\n")
        binary.chmod(0o755)
        self.token = self.home / ".config/second-opinion/mcp-token"
        self.token.parent.mkdir(parents=True)
        self.env = {"HOME": str(self.home), "PATH": "/usr/bin:/bin"}

    def run_launcher(self, data=b"a" * 64 + b"\n", mode=0o600):
        if data is not None:
            self.token.write_bytes(data)
            self.token.chmod(mode)
        return subprocess.run([str(self.launcher)], env=self.env, capture_output=True, text=True, timeout=3)

    @unittest.skipIf(os.getuid() == 0, "service intentionally refuses root")
    def test_exec_arguments_environment_and_status(self):
        result = self.run_launcher()
        self.assertEqual(result.returncode, 17, result.stderr)
        lines = result.stdout.splitlines()
        self.assertEqual(lines[0], "a" * 64)
        self.assertTrue(lines[1].startswith(str(self.home / ".local/bin") + ":"))
        self.assertEqual(lines[2:], ["mcp", "serve", "--transport", "http", "--listen", "127.0.0.1:8765"])
        self.assertEqual(result.stderr, "")
        self.env["SECOND_OPINION_MCP_LISTEN"] = "[::1]:9876"
        self.assertEqual(self.run_launcher(b"b" * 32, 0o400).stdout.splitlines()[-1], "[::1]:9876")

    def test_invalid_configuration_never_executes_or_leaks(self):
        for data, mode in [(None, 0o600), (b"secret" * 8, 0o644), (b"short", 0o600),
                           (b"a" * 4098, 0o600), (b"a" * 64 + b"\n\n", 0o600),
                           (b"a" * 64 + b"\x00", 0o600), (b"a" * 64 + b" b", 0o600)]:
            with self.subTest(data_len=len(data) if data else 0, mode=mode):
                result = self.run_launcher(data, mode)
                self.assertEqual(result.returncode, 1)
                self.assertEqual(result.stdout, "")
                self.assertNotIn("secret", result.stderr)
        self.token.unlink()
        self.token.symlink_to(self.home / "bin/second-opinion")
        self.assertEqual(self.run_launcher(None).returncode, 1)

    @unittest.skipIf(os.getuid() == 0, "service intentionally refuses root")
    def test_explicit_token_path(self):
        self.env["SECOND_OPINION_MCP_TOKEN_FILE"] = str(self.home / "custom-token")
        Path(self.env["SECOND_OPINION_MCP_TOKEN_FILE"]).write_text("x" * 64)
        Path(self.env["SECOND_OPINION_MCP_TOKEN_FILE"]).chmod(0o600)
        self.assertEqual(self.run_launcher(None).returncode, 17)


class FormulaTests(unittest.TestCase):
    def test_render(self):
        result = subprocess.run([str(ROOT / "scripts/render-homebrew-formula"), "v0.2.1", "a" * 64], capture_output=True, text=True, check=True)
        self.assertIn("v0.2.1.tar.gz", result.stdout)
        self.assertIn('sha256 "' + "a" * 64 + '"', result.stdout)
        self.assertIn('run [opt_libexec/"second-opinion-mcp-service"]', result.stdout)
        self.assertNotIn("@TAG@", result.stdout)
        self.assertNotIn("SECOND_OPINION_MCP_TOKEN=", result.stdout)
        invalid = subprocess.run([str(ROOT / "scripts/render-homebrew-formula"), 'v1.2.3"', "a" * 64], capture_output=True)
        self.assertNotEqual(invalid.returncode, 0)
        self.assertEqual(invalid.stdout, b"")
