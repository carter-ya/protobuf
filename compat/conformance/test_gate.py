import hashlib
import unittest
from run import check_results, observe


class GateTests(unittest.TestCase):
    def setUp(self):
        self.tests = {'Required.Proto3.Pass', 'Required.Proto3.Fail', 'Required.Proto3.Variable', 'Required.Future.Skip'}
        self.expected = {'tests': 4, 'inventory_sha256': hashlib.sha256('\n'.join(sorted(self.tests)).encode()).hexdigest(),
                         'failures': [{'test': 'Required.Proto3.Fail'}],
                         'variable_outcomes': [{'test': 'Required.Proto3.Variable'}],
                         'skips': [{'test': 'Required.Future.Skip'}]}

    def check(self, failures=None, skips=None, tests=None, summaries=None, code=1):
        return check_results('suite', self.tests if tests is None else tests,
                             {'Required.Proto3.Fail': 'known'} if failures is None else failures,
                             {'Required.Future.Skip'} if skips is None else skips,
                             ['CONFORMANCE SUITE FAILED: complete'] if summaries is None else summaries,
                             code, self.expected)

    def test_known_failure_and_variable_outcome(self):
        self.assertEqual([], self.check())
        self.assertEqual([], self.check(failures={'Required.Proto3.Fail': 'known', 'Required.Proto3.Variable': 'known nondeterminism'}))

    def test_new_failure_or_skip_cannot_pass(self):
        self.assertTrue(self.check(failures={'Required.Proto3.Fail': 'known', 'Required.Proto3.Pass': 'regression'}))
        self.assertTrue(self.check(skips={'Required.Future.Skip', 'Required.Proto3.Pass'}))

    def test_improvements_require_removing_baseline(self):
        self.assertTrue(self.check(failures={}))

    def test_missing_tests_or_crashed_runner_cannot_pass(self):
        self.assertTrue(self.check(tests=set()))
        self.assertTrue(self.check(summaries=[]))
        self.assertTrue(self.check(code=-9))

    def test_log_observation(self):
        log = ('conformance test: name=Required.Proto3.Bad, request=x\n'
               'ERROR, test=Required.Proto3.Bad: invalid output, request=x\n'
               'SKIPPED, test=Required.Future.Skip request=x\n'
               'CONFORMANCE SUITE FAILED: 1 unexpected failures.\n')
        tests, failures, skips, summaries = observe(log)
        self.assertEqual({'Required.Proto3.Bad'}, tests)
        self.assertEqual({'Required.Proto3.Bad': 'invalid output'}, failures)
        self.assertEqual({'Required.Future.Skip'}, skips)
        self.assertEqual(1, len(summaries))


if __name__ == '__main__':
    unittest.main()
