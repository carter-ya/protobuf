#!/usr/bin/env python3
"""Inventory MessageSet declarations in the checksum-pinned downstream archives."""
import argparse
import hashlib
import json
import pathlib
import subprocess

HERE = pathlib.Path(__file__).resolve().parent


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=pathlib.Path, required=True)
    args = parser.parse_args()
    rows = []
    for project in json.loads((HERE / 'pins.json').read_text()):
        for pin in project['modules']:
            data = json.loads(subprocess.check_output(['go', 'mod', 'download', '-json', pin['Path'] + '@' + pin['Version']], text=True))
            for key in ['Path', 'Version', 'Sum', 'GoModSum']:
                if data[key] != pin[key]:
                    raise ValueError('downstream archive identity drift: ' + key)
            directory = pathlib.Path(data['Dir'])
            schemas = sorted(directory.rglob('*.proto'))
            inventory = []
            hits = []
            for path in schemas:
                raw = path.read_bytes()
                name = path.relative_to(directory).as_posix()
                inventory.append({'path': name, 'sha256': hashlib.sha256(raw).hexdigest()})
                if b'message_set_wire_format' in raw:
                    hits.append(name)
            rows.append({'project': project['name'], 'module': pin, 'schema_files_scanned': len(schemas),
                         'schema_inventory_sha256': hashlib.sha256(json.dumps(inventory, sort_keys=True).encode()).hexdigest(),
                         'message_set_option_occurrences': hits})
    report = {'scope': 'All .proto files delivered in the five pinned Go module archives; excludes other versions, omitted repository files and deployed workloads.',
              'rows': rows, 'decision': 'Preserve legacy MessageSet support. Defer modern ProtoReflect/HYBRID/OPAQUE MessageSet until a concrete downstream schema and workload require it.'}
    if any(row['message_set_option_occurrences'] for row in rows):
        raise ValueError('MessageSet demand evidence changed; review the modern-support decision')
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print('Verified MessageSet inventory:', sum(row['schema_files_scanned'] for row in rows), 'schemas, no occurrences.')


if __name__ == '__main__':
    main()
