// This file contains tests for control file rendering. They are in a separate
// file because they are somewhat verbose.

package deb

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"testing"

)

func TestRenderControlFileBasic(t *testing.T) {
	p := readJSON("example-basic.json")
	p.Version = "0.1.0"
	p.Section = "default"

	expected := `Package: mkdeb
Version: 0.1.0
Architecture: amd64
Maintainer: Chris Bednarski <banzaimonkey@gmail.com>
Installed-Size: 12345
Section: default
Priority: optional
Homepage: https://github.com/cbednarski/mkdeb
Description: A CLI tool for building debian packages
`
	var b bytes.Buffer
	if err := p.WriteControl(&b, 12345); err != nil {
		t.Fatal(err)
	}
	if b.String() != expected {
		t.Fatalf("Control file did not match expected\n%s\n--Found--\n%s\n", expected, b.String())
	}
}

func TestRenderControlFileWithDepends(t *testing.T) {
	p := readJSON("example-depends.json")

	p.Conflicts = []string{}
	p.Version = "0.1.0"

	expected := `Package: mkdeb
Version: 0.1.0
Architecture: amd64
Maintainer: Chris Bednarski <banzaimonkey@gmail.com>
Installed-Size: 0
Depends: wget, tree
Priority: optional
Homepage: https://github.com/cbednarski/mkdeb
Description: A CLI tool for building debian packages
`
	var b bytes.Buffer
	if err := p.WriteControl(&b, 0); err != nil {
		t.Fatal(err)
	}
	if b.String() != expected {
		t.Fatalf("Control file did not match expected\n%s\n--Found--\n%s\n", expected, b.String())
	}
}

func TestRenderControlFileWithPreDepends(t *testing.T) {
	p := readJSON("example-predepends.json")
	p.Conflicts = []string{}
	p.Version = "0.1.0"

	expected := `Package: mkdeb
Version: 0.1.0
Architecture: amd64
Maintainer: Chris Bednarski <banzaimonkey@gmail.com>
Installed-Size: 0
Pre-Depends: wget, tree
Priority: optional
Homepage: https://github.com/cbednarski/mkdeb
Description: A CLI tool for building debian packages
`
	var b bytes.Buffer
	if err := p.WriteControl(&b, 0); err != nil {
		t.Fatal(err)
	}
	if b.String() != expected {
		t.Fatalf("Control file did not match expected\n%s\n--Found--\n%s\n", expected, b.String())
	}
}

func TestRenderControlFileWithReplaces(t *testing.T) {
	p := readJSON("example-replaces.json")

	p.Replaces = []string{"debpkg"}
	p.Version = "0.1.0"

	expected := `Package: mkdeb
Version: 0.1.0
Architecture: amd64
Maintainer: Chris Bednarski <banzaimonkey@gmail.com>
Installed-Size: 0
Depends: wget, tree
Conflicts: debpkg
Replaces: debpkg
Priority: optional
Homepage: https://github.com/cbednarski/mkdeb
Description: A CLI tool for building debian packages
`
	var b bytes.Buffer
	if err := p.WriteControl(&b, 0); err != nil {
		t.Fatal(err)
	}
	if b.String() != expected {
		t.Fatalf("Control file did not match expected\n%s\n--Found--\n%s\n", expected, b.String())
	}
}

func readJSON(filename string) Control {
	data, err := ioutil.ReadFile(path.Join("test-fixtures", filename))
	if err != nil {
		log.Fatal(err)
	}
	var c Control
	if err := json.Unmarshal(data, &c); err != nil {
		log.Fatal(err)
	}
	return c
}
