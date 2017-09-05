package deb

import "testing"

var (
	exampleBasic = Control{
		Architecture: "amd64",
		Maintainer:   "Chris Bednarski <banzaimonkey@gmail.com>",
		Package:      "mkdeb",
		Version:      "0.1.0",
		Homepage:     "https://github.com/cbednarski/mkdeb",
		Description:  "A CLI tool for building debian packages",
	}
)

func TestFilename(t *testing.T) {
	c := Control{
		Package:      "mkdeb",
		Version:      "0.1.0",
		Architecture: "amd64",
	}
	expected := "mkdeb-0.1.0-amd64.deb"
	if c.debFilename() != expected {
		t.Fatalf("Expected filename to be %q, got %q", expected, c.debFilename())
	}
}

func TestValidate(t *testing.T) {
	{
		c := exampleBasic
		if err := c.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	{
		c := Control{}
		err := c.Validate()
		expected := "These required fields are missing: package, version, architecture, maintainer, description"
		if err.Error() != expected {
			t.Fatalf("-- Expected --\n%s\n-- Found --\n%s\n", expected, err.Error())
		}
	}
}
