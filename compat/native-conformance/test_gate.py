import importlib.util
import pathlib
import unittest

spec = importlib.util.spec_from_file_location('gate', pathlib.Path(__file__).with_name('run.py'))
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)


class TestExactGate(unittest.TestCase):
    def test_rejects_missing_tests_new_failures_and_skips(self):
        names = ['pass', 'known']
        observed = ({'pass', 'known'}, {'known': 'reason'}, set(), ['CONFORMANCE SUITE FAILED'], 1)
        self.assertTrue(gate.validate(names, observed, ['known']))
        for bad in [({'pass'}, observed[1], set(), observed[3], 1),
                    (observed[0], {'known': 'reason', 'pass': 'new'}, set(), observed[3], 1),
                    (observed[0], observed[1], {'pass'}, observed[3], 1),
                    (observed[0], {}, set(), observed[3], 0),
                    (observed[0], observed[1], set(), [], 1),
                    (observed[0], observed[1], set(), observed[3], 2)]:
            self.assertFalse(gate.validate(names, bad, ['known']))


if __name__ == '__main__':
    unittest.main()
