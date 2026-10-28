#!/usr/bin/env python3
"""Add modern protocol declarations without removing historical public fields.

Usage: python3 compat/merge-protocol.py /path/to/protobuf-36.2/include
Only optional/repeated fields and message/enum declarations are merged. Existing
declarations, field numbers, defaults and source options remain authoritative.
CLI output is formatted with clang-format (Google style, preserving comments).
Set CLANG_FORMAT to use a specific formatter binary.
"""
import os
import pathlib
import subprocess
import re
import sys

TOKEN = re.compile(r'//[^\n]*|/\*[\s\S]*?\*/|"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'')
DECL = re.compile(r'\b(message|enum)\s+(\w+)\s*\{')
FIELD = re.compile(r'\b(optional|required|repeated)\s+[.\w]+\s+(\w+)\s*=\s*(\d+)')
ENUM_VALUE = re.compile(r'\b([A-Z][A-Z_0-9]*)\s*=\s*(-?\d+)')


class Source:
    def __init__(self, text):
        self.text = text
        self.mask = TOKEN.sub(lambda m: re.sub(r'[^\n]', ' ', m.group()), text)
        self.depth = []
        level = 0
        for char in self.mask:
            self.depth.append(level)
            level += (char == '{') - (char == '}')
        if level:
            raise ValueError('unbalanced proto braces')

    def declarations(self, start=0, end=None, depth=0):
        end = len(self.text) if end is None else end
        result = {}
        for match in DECL.finditer(self.mask, start, end):
            if self.depth[match.start()] != depth:
                continue
            opening = match.end() - 1
            closing = next(i for i in range(opening + 1, end)
                           if self.mask[i] == '}' and self.depth[i] == depth + 1)
            result[match.group(1), match.group(2)] = (match.start(), opening, closing)
        return result

    def fields(self, opening, closing, enum=False):
        result = {}
        for match in (ENUM_VALUE if enum else FIELD).finditer(self.mask, opening + 1, closing):
            if self.depth[match.start()] != self.depth[opening] + 1:
                continue
            end = self.mask.index(';', match.end()) + 1
            result[int(match.group(2 if enum else 3))] = (match.group(1 if enum else 2), self.text[match.start():end])
        return result


def merge(old_text, upstream_text):
    old, upstream = Source(old_text), Source(upstream_text)
    edits = []

    def visit(old_nodes, new_nodes, parent_end, depth):
        additions = []
        for identity, new_node in new_nodes.items():
            if identity not in old_nodes:
                additions.append(upstream.text[new_node[0]:new_node[2] + 1])
                continue
            old_node = old_nodes[identity]
            enum = identity[0] == 'enum'
            old_fields = old.fields(old_node[1], old_node[2], enum)
            old_names = {field[0] for field in old_fields.values()}
            fields = [field[1] for number, field in upstream.fields(new_node[1], new_node[2], enum).items()
                      if number not in old_fields and field[0] not in old_names]
            if fields:
                edits.append((old_node[2], '\n' + '\n'.join('  ' + f for f in fields) + '\n'))
            visit(old.declarations(old_node[1] + 1, old_node[2], depth + 1),
                  upstream.declarations(new_node[1] + 1, new_node[2], depth + 1), old_node[2], depth + 1)
        if additions:
            edits.append((parent_end, '\n\n' + '\n\n'.join(additions) + '\n'))

    visit(old.declarations(), upstream.declarations(), len(old_text), 0)
    result = old_text
    for position, addition in sorted(edits, reverse=True):
        result = result[:position] + addition + result[position:]
    return '\n'.join(line.rstrip() for line in result.splitlines()) + '\n'


if __name__ == '__main__':
    include = pathlib.Path(sys.argv[1])
    for relative in ['google/protobuf/descriptor.proto', 'google/protobuf/compiler/plugin.proto']:
        target = pathlib.Path('protobuf') / relative
        merged = merge(target.read_text(), (include / relative).read_text())
        formatted = subprocess.run(
            [os.environ.get('CLANG_FORMAT', 'clang-format'),
             '--style={BasedOnStyle: Google, ReflowComments: false}',
             '--assume-filename=' + target.name],
            input=merged, text=True, stdout=subprocess.PIPE, check=True)
        target.write_text(formatted.stdout)
