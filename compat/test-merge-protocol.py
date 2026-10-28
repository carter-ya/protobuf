import importlib.util
import pathlib
import subprocess
import unittest

spec = importlib.util.spec_from_file_location('merge_protocol', pathlib.Path(__file__).with_name('merge-protocol.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class ProtocolMergeTest(unittest.TestCase):
    def test_additions_preserve_legacy_fields_and_defaults(self):
        old = 'message A { optional bool old_field = 1 [default=false]; message B { optional string value=1; } }\nenum E { OLD = 0; }\n'
        upstream = 'message A { optional bool old_field = 1 [default=true]; optional int32 added=2; message B { optional string value=1; optional int32 added=2; } }\nenum E { OLD = 0; NEW = 1; }\nmessage NewType { optional string value=1; }\n'
        result = module.merge(old, upstream)
        self.assertIn('old_field = 1 [default=false]', result)
        self.assertNotIn('default=true', result)
        self.assertEqual(result.count('optional int32 added=2'), 2)
        self.assertIn('NEW = 1', result)
        self.assertIn('message NewType', result)
        self.assertEqual(module.merge(result, upstream), result)

    def test_all_original_fields_and_enum_values_survive(self):
        for path in ['protobuf/google/protobuf/descriptor.proto', 'protobuf/google/protobuf/compiler/plugin.proto']:
            old = module.Source(subprocess.check_output(['git', 'show', 'v1.3.2:' + path], text=True))
            current = module.Source(pathlib.Path(path).read_text())

            def check(old_nodes, new_nodes, depth):
                for identity, node in old_nodes.items():
                    self.assertIn(identity, new_nodes)
                    target = new_nodes[identity]
                    old_fields = old.fields(node[1], node[2], identity[0] == 'enum')
                    new_fields = current.fields(target[1], target[2], identity[0] == 'enum')
                    for number, field in old_fields.items():
                        self.assertEqual(new_fields[number], field, (path, identity, number))
                    check(old.declarations(node[1]+1,node[2],depth+1), current.declarations(target[1]+1,target[2],depth+1), depth+1)
            check(old.declarations(), current.declarations(), 0)


if __name__ == '__main__':
    unittest.main()
