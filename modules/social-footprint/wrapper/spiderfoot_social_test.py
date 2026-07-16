#!/usr/bin/env python3
"""Minimal offline tests for spiderfoot_social.py.

These tests exercise the wrapper's CLI contract and its graceful import-failure
path without requiring a real SpiderFoot installation or network access.
"""
import json
import os
import subprocess
import sys
import tempfile
import unittest


class TestSpiderFootSocialWrapper(unittest.TestCase):
    def run_wrapper(self, extra_env=None):
        """Run spiderfoot_social.py with a stub SpiderFoot root that cannot be
        imported, returning (stdout, stderr, returncode)."""
        with tempfile.TemporaryDirectory() as tmp:
            # Create a fake SpiderFoot root whose sflib/spiderfoot raise on import.
            os.makedirs(os.path.join(tmp, "spiderfoot"), exist_ok=True)
            with open(os.path.join(tmp, "sflib.py"), "w") as f:
                f.write("raise ImportError('stub: sflib unavailable')\n")
            with open(os.path.join(tmp, "spiderfoot", "__init__.py"), "w") as f:
                f.write("raise ImportError('stub: spiderfoot unavailable')\n")
            os.makedirs(os.path.join(tmp, "modules"), exist_ok=True)
            with open(os.path.join(tmp, "modules", "__init__.py"), "w") as f:
                pass

            env = os.environ.copy()
            env["SPIDERFOOT_ROOT"] = tmp
            env["SOCIAL_FOOTPRINT_SPIDERFOOT_ROOT"] = tmp
            if extra_env:
                env.update(extra_env)

            script = os.path.join(os.path.dirname(__file__), "spiderfoot_social.py")
            proc = subprocess.run(
                [sys.executable, script, "--username", "testuser",
                 "--sites", "GitHub,Keybase"],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                env=env,
            )
            return proc

    def test_import_failure_returns_json_and_exit_zero(self):
        proc = self.run_wrapper()
        self.assertEqual(proc.returncode, 0,
                         f"expected exit 0 for recoverable import failure, got {proc.returncode}\nstderr: {proc.stderr}")
        data = json.loads(proc.stdout)
        self.assertEqual(data["handle"], "testuser")
        self.assertEqual(data["sites_requested"], ["GitHub", "Keybase"])
        self.assertIn("spiderfoot", data["error"].lower())
        self.assertEqual(data["claimed_count"], 0)

    def test_bad_arguments_exit_nonzero(self):
        script = os.path.join(os.path.dirname(__file__), "spiderfoot_social.py")
        proc = subprocess.run(
            [sys.executable, script],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        self.assertNotEqual(proc.returncode, 0)


if __name__ == "__main__":
    unittest.main()
