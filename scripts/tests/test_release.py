import os
from pathlib import Path
import subprocess
import tempfile
import unittest

CHECK = Path(__file__).resolve().parents[1] / "check-release"


class ReleaseChecks(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.remote = self.root / "remote.git"
        self.repo = self.root / "work"
        self.env = dict(os.environ, GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull,
                        GIT_AUTHOR_NAME="Test", GIT_AUTHOR_EMAIL="test@example.invalid",
                        GIT_COMMITTER_NAME="Test", GIT_COMMITTER_EMAIL="test@example.invalid")
        self.git("init", "--bare", "--initial-branch=main", str(self.remote), cwd=self.root)
        self.git("clone", str(self.remote), str(self.repo), cwd=self.root)
        version = self.repo / "internal/buildinfo/VERSION"
        version.parent.mkdir(parents=True)
        version.write_text("1.0.0\n")
        self.commit("initial")
        self.git("push", "origin", "main")

    def git(self, *args, cwd=None, check=True):
        return subprocess.run(["git", *args], cwd=cwd or self.repo, env=self.env,
                              text=True, capture_output=True, check=check)

    def commit(self, message):
        self.git("add", ".")
        self.git("commit", "--allow-empty", "-m", message)

    def guard(self, mode="prepare", tag="v1.0.0", success=False):
        before = self.git("rev-parse", "HEAD").stdout
        result = subprocess.run([str(CHECK), mode, tag], cwd=self.repo, env=self.env,
                                text=True, capture_output=True)
        self.assertEqual(result.returncode == 0, success, result.stderr)
        self.assertEqual(self.git("rev-parse", "HEAD").stdout, before)
        return result

    def test_preparation_requires_clean_synchronized_main(self):
        self.guard(success=True)
        self.git("switch", "-c", "feature")
        self.guard()
        self.git("switch", "main")
        (self.repo / "untracked").write_text("dirty")
        self.guard()
        (self.repo / "untracked").unlink()
        self.commit("ahead")
        self.guard()
        self.git("push", "origin", "main")
        self.guard(success=True)
        self.git("reset", "--hard", "HEAD~1")
        self.guard()  # behind
        self.commit("different")
        self.guard()  # diverged

    def test_existing_tags_and_invalid_versions(self):
        self.guard(tag="v1.0.0;echo nope")
        self.git("tag", "v1.0.0")
        self.guard()
        self.git("push", "origin", "v1.0.0")
        self.git("tag", "-d", "v1.0.0")
        self.guard()  # remote tag is sufficient to block reuse

    def test_publication_ancestry_identity_and_version(self):
        self.git("tag", "v1.0.0")
        self.guard("publish", success=True)
        self.git("switch", "-c", "feature")
        self.commit("off main")
        self.git("tag", "v1.0.1")
        self.guard("publish", "v1.0.1")
        self.guard("publish", "v1.0.0")  # wrong checkout
        self.git("push", "origin", "HEAD:main")
        self.guard("publish", "v1.0.1")  # reachable but wrong embedded version
        (self.repo / "internal/buildinfo/VERSION").write_text("1.0.2\n")
        self.commit("version")
        self.git("tag", "v1.0.2")
        self.git("push", "origin", "HEAD:main")
        self.guard("publish", "v1.0.2", success=True)

    def test_concurrent_remote_advance_rejects_atomic_tag(self):
        self.guard(success=True)
        other = self.root / "other"
        self.git("clone", str(self.remote), str(other), cwd=self.root)
        self.git("commit", "--allow-empty", "-m", "remote advance", cwd=other)
        self.git("push", "origin", "main", cwd=other)
        self.commit("release")
        self.git("tag", "v1.0.0")
        result = self.git("push", "--atomic", "origin", "HEAD:refs/heads/main", "refs/tags/v1.0.0", check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.git("ls-remote", "--tags", "origin", "refs/tags/v1.0.0").stdout, "")
