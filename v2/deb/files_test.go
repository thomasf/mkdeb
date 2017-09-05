package deb

import (
	"testing"

	deb1 "github.com/cbednarski/mkdeb/deb"
)

func TestBuildDeb(t *testing.T) {
	ps := deb1.DefaultPackageSpec()
	ps.AutoPath = "test-fixtures/package1"
	ps.Files = map[string]string{
		"test-fixtures/example-replaces.json": "dangus/example-replaces-renamed.json",
	}
	fs := fromPackageSpec(ps)
	c := newControlFromPackageSpec(*ps)
	c.Version = "1.0.0"
	c.Package = "asdfd"
	c.Architecture = "amd64"
	c.Description = "asdasd"
	c.Maintainer = "asd"

	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := createDeb(".", fs, c); err != nil {
		t.Fatal(err)
	}
}
