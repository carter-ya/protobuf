import importlib.util
import pathlib
import subprocess
import tempfile
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location('build_runner', pathlib.Path(__file__).with_name('build-runner.py'))
build = importlib.util.module_from_spec(spec)
spec.loader.exec_module(build)


class CachedRunnerTests(unittest.TestCase):
    def test_missing_or_broken_cached_runner_is_not_ready(self):
        with tempfile.TemporaryDirectory() as directory:
            executable = pathlib.Path(directory) / 'runner'
            self.assertFalse(build.runner_ready(executable))
            executable.touch()
            with mock.patch.object(build.subprocess, 'run', return_value=subprocess.CompletedProcess([], 127)):
                self.assertFalse(build.runner_ready(executable))
            with mock.patch.object(build.subprocess, 'run', side_effect=OSError('loader failed')):
                self.assertFalse(build.runner_ready(executable))
            with mock.patch.object(build.subprocess, 'run', side_effect=subprocess.TimeoutExpired([], 10)):
                self.assertFalse(build.runner_ready(executable))
            with mock.patch.object(build.subprocess, 'run', return_value=subprocess.CompletedProcess([], 2)):
                self.assertTrue(build.runner_ready(executable))


if __name__ == '__main__':
    unittest.main()
