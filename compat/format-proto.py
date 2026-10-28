#!/usr/bin/env python3
"""Format proto files added or changed since the original maintenance baseline."""
import argparse
import difflib
import os
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]


def git_paths(*arguments):
    output = subprocess.check_output(['git'] + list(arguments), cwd=ROOT)
    return {os.fsdecode(path) for path in output.split(b'\0') if path}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true', help='report formatting differences without changing files')
    parser.add_argument('--base', default='v1.3.2', help='Git baseline for selecting maintained proto files (default: v1.3.2)')
    args = parser.parse_args()
    paths = git_paths('diff', '--name-only', '--diff-filter=ACMR', '-z', args.base, '--', '*.proto')
    paths.update(git_paths('ls-files', '--others', '--exclude-standard', '-z', '--', '*.proto'))
    formatter = os.environ.get('CLANG_FORMAT', 'clang-format')
    required = (ROOT / 'compat/proto-format-requirements.txt').read_text().strip().split('==')[1]
    version = subprocess.check_output([formatter, '--version'], text=True)
    found = re.search(r'\bversion\s+(\d+\.\d+\.\d+)\b', version)
    if found is None or found.group(1) != required:
        sys.exit('Expected clang-format %s; install compat/proto-format-requirements.txt in a virtual environment and set CLANG_FORMAT.' % required)
    changed = 0
    for name in sorted(paths):
        path = ROOT / name
        original = path.read_bytes()
        formatted = subprocess.check_output([
            formatter, '--style=file:' + str(ROOT / '.clang-format'),
            '--assume-filename=' + str(path),
        ], input=original)
        if formatted == original:
            continue
        changed += 1
        if args.check:
            sys.stdout.writelines(difflib.unified_diff(
                original.decode('utf-8').splitlines(keepends=True),
                formatted.decode('utf-8').splitlines(keepends=True),
                fromfile=name, tofile=name + ' (formatted)',
            ))
        else:
            path.write_bytes(formatted)
            print('Formatted ' + name)
    if args.check and changed:
        sys.exit('Proto formatting differs; run python3 compat/format-proto.py with the pinned formatter.')
    print('%s %d proto files; %d %s.' % (
        'Checked' if args.check else 'Processed', len(paths), changed,
        'need formatting' if args.check else 'formatted',
    ))


if __name__ == '__main__':
    main()
