#!/usr/bin/env python3
"""Build the pinned upstream suites with an independent-suite entrypoint."""
import argparse
import hashlib
import pathlib
import shutil
import subprocess
import tarfile
import urllib.request

VERSION = '36.2'
SHA256 = '3d9642a662d10e68ebae5e53f14dcce5105684212d5078f8e0d47d1ab3ae6b64'
URL = 'https://github.com/protocolbuffers/protobuf/releases/download/v36.2/protobuf-36.2.tar.gz'


def runner_ready(executable):
    if not executable.is_file():
        return False
    try:
        # Our entrypoint returns 2 before constructing a suite when no suite is
        # selected. The OS loader must still resolve every linked library.
        result = subprocess.run([str(executable)], stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, timeout=10)
    except (OSError, subprocess.TimeoutExpired):
        return False
    return result.returncode == 2


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory', type=pathlib.Path)
    parser.add_argument('--jobs', default='2')
    args = parser.parse_args()
    root = args.directory.resolve()
    root.mkdir(parents=True, exist_ok=True)
    archive = root / 'protobuf-36.2.tar.gz'
    if not archive.exists():
        with urllib.request.urlopen(URL, timeout=120) as response:
            archive.write_bytes(response.read())
    if hashlib.sha256(archive.read_bytes()).hexdigest() != SHA256:
        raise SystemExit('upstream archive checksum mismatch')
    source = root / 'protobuf-36.2'
    if not source.exists():
        with tarfile.open(archive) as stream:
            stream.extractall(root, filter='data')
    executable = root / 'bin/conformance_test_runner'
    if executable.exists() and not runner_ready(executable):
        print('Cached conformance runner cannot start; rebuilding it.')
        executable.unlink()
    if not executable.exists():
        # No upstream protocol, fixture or test implementation is modified.
        shutil.copyfile(pathlib.Path(__file__).with_name('runner-main.cc'),
                        source / 'conformance/conformance_test_main.cc')
        # Upstream's static-link branch uses jsoncpp_static, but its dependency
        # check still requires jsoncpp_lib. Supply an alias for the static-only
        # build; no test implementation, schema or protocol is changed.
        cmake = source / 'cmake/conformance.cmake'
        config = cmake.read_text()
        marker = 'if (NOT TARGET jsoncpp_lib)\n  message(FATAL_ERROR'
        alias = ('if (TARGET jsoncpp_static AND NOT TARGET jsoncpp_lib)\n'
                 '  add_library(jsoncpp_lib ALIAS jsoncpp_static)\nendif()\n\n')
        if alias not in config:
            if config.count(marker) != 1:
                raise SystemExit('pinned conformance CMake target check changed')
            cmake.write_text(config.replace(marker, alias + marker))
        build = root / 'build'
        subprocess.run(['cmake', '-S', str(source), '-B', str(build), '-G', 'Ninja',
                        '-DCMAKE_BUILD_TYPE=Release', '-DBUILD_SHARED_LIBS=OFF',
                        '-Dprotobuf_BUILD_SHARED_LIBS=OFF', '-DBUILD_STATIC_LIBS=ON',
                        '-Dprotobuf_BUILD_TESTS=OFF',
                        '-Dprotobuf_BUILD_CONFORMANCE=ON', '-Dprotobuf_INSTALL=OFF',
                        '-Dprotobuf_FORCE_FETCH_DEPENDENCIES=ON', '-Dprotobuf_LOCAL_DEPENDENCIES_ONLY=OFF'], check=True)
        subprocess.run(['cmake', '--build', str(build), '--target', 'conformance_test_runner', '-j', args.jobs], check=True)
        executable.parent.mkdir(exist_ok=True)
        shutil.copy2(build / 'conformance_test_runner', executable)
    if not runner_ready(executable):
        raise SystemExit('built conformance runner cannot start')
    print(str(executable))


if __name__ == '__main__':
    main()
