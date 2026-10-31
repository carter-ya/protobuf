"""Exercise compiler-owned Edition naming, visibility and size boundaries."""
import pathlib
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
out = pathlib.Path(sys.argv[1]).resolve()
out.mkdir()
cases = 0

def check(name, body, accepted, marker=None, edition='2026'):
    global cases
    source = out / (name + '.proto')
    source.write_text('edition="' + edition + '";package audit;option go_package="example.com/audit/pb";\n' + body)
    result = subprocess.run(['protoc', '-I' + str(out), '--descriptor_set_out=' + str(out / (name + '.pb')), source.name],
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    if (result.returncode == 0) != accepted or (not accepted and marker and marker not in result.stdout):
        raise RuntimeError(name + ': unexpected compiler result: ' + result.stdout)
    cases += 1

check('style2023', 'message Foo {int32 bar_1=1;}', True, edition='2023')
check('style2024', 'message Foo {int32 bar_1=1;}', False, edition='2024')
check('style_legacy', 'option features.enforce_naming_style=STYLE_LEGACY;message Foo {int32 bar_1=1;}', True, edition='2024')
check('collision2026', 'message Foo {int32 value=1;int32 has_value=2;}', False)
check('collision_override', 'option features.enforce_naming_style=STYLE2024;message Foo {int32 value=1;int32 has_value=2;}', True)
(out / 'visibility.proto').write_text('edition="2026";package audit;option go_package="example.com/audit/pb";local message Hidden {} export message Visible {}')
check('visibility_ok', 'import "visibility.proto";message Record {Visible value=1;}', True)
check('visibility_bad', 'import "visibility.proto";message Record {Hidden value=1;}', False)

for kind, maximum in [('fields', 1500), ('oneofs', 1000), ('members', 1200), ('enum', 1700)]:
    for count, override, accepted in [(maximum, False, True), (maximum + 1, False, False), (maximum + 1, True, True)]:
        option = 'option features.enforce_proto_limits=LEGACY_NO_EXPLICIT_LIMITS;' if override else ''
        fields = ''.join('int32 value%d=%d;' % (i, i) for i in range(1, count + 1))
        if kind == 'fields':
            body = 'message Record {' + option + fields + '}'
        elif kind == 'oneofs':
            body = 'message Record {' + option + ''.join('oneof choice%d {int32 value%d=%d;}' % (i, i, i) for i in range(1, count + 1)) + '}'
        elif kind == 'members':
            body = 'message Record {oneof choice {' + option + fields + '}}'
        else:
            body = 'enum Mode {' + option + ''.join('MODE_VALUE%d=%d;' % (i, i) for i in range(count)) + '}'
        check(kind + str(count) + ('_override' if override else ''), body, accepted, str(maximum))

# Pass accepted multi-file descriptors through the real generator, not just protoc.
driver = out / 'gogo'
subprocess.run(['go', 'build', '-o', driver, './protoc-gen-gogo'], cwd=ROOT, check=True)
subprocess.run(['protoc', '-I' + str(out), '--plugin=protoc-gen-test=' + str(driver),
                '--test_out=paths=source_relative:' + str(out), 'visibility.proto', 'visibility_ok.proto', 'collision_override.proto'], check=True)
print('Compiler-owned feature boundary cases passed:', cases)
