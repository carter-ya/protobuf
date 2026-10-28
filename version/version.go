// Protocol Buffers for Go with Gadgets
//
// Copyright (c) 2013, The GoGo Authors. All rights reserved.
// http://github.com/gogo/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package version

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func Get() string {
	versionBytes, _ := exec.Command("protoc", "--version").CombinedOutput()
	version := strings.TrimSpace(string(versionBytes))
	versions := strings.Split(version, " ")
	if len(versions) != 2 {
		panic("version string returned from protoc is separated with a space: " + version)
	}
	return versions[1]
}

func parseVersion(version string) ([3]int, error) {
	var result [3]int
	versions := strings.Split(version, ".")
	if len(versions) != 2 && len(versions) != 3 {
		return result, fmt.Errorf("version must have 2 or 3 numbers separated by dots: %s", version)
	}
	for index, v := range versions {
		if v == "" || strings.IndexFunc(v, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			return result, fmt.Errorf("invalid version component %q", v)
		}
		i, err := strconv.Atoi(v)
		if err != nil {
			return result, err
		}
		result[index] = i
	}
	return result, nil
}

func less(this, that string) bool {
	thisNum, err := parseVersion(this)
	if err != nil {
		panic(err)
	}
	thatNum, err := parseVersion(that)
	if err != nil {
		panic(err)
	}
	for i := range thisNum {
		if thisNum[i] != thatNum[i] {
			return thisNum[i] < thatNum[i]
		}
	}
	return true
}

func AtLeast(v string) bool {
	return less(v, Get())
}
