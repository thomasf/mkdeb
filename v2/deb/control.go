package deb

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	deb1 "github.com/cbednarski/mkdeb/deb"
	"github.com/cbednarski/mkdeb/deb/tar"
	"github.com/klauspost/pgzip"
	"github.com/thomasf/vfs"
)

// Control represents the debian binary control file
type Control struct {
	// Binary Debian Control File - Required fields
	Package      string `json:"package"`
	Version      string `json:"-"`
	Architecture string `json:"architecture"`
	Maintainer   string `json:"maintainer"`
	Description  string `json:"description"`

	// Optional Fields
	Depends    []string `json:"depends"`
	PreDepends []string `json:"preDepends"`
	Conflicts  []string `json:"conflicts,omitempty"`
	Breaks     []string `json:"breaks,omitempty"`
	Replaces   []string `json:"replaces,omitempty"`
	Section    string   `json:"section"`
	Priority   string   `json:"priority"` // Defaults to "optional"
	Homepage   string   `json:"homepage"`

}

func newControlFromPackageSpec(p deb1.PackageSpec) Control {
	return Control{
		Package:      p.Package,
		Version:      p.Version,
		Architecture: p.Architecture,
		Maintainer:   p.Maintainer,
		Description:  p.Description,
		Depends:      p.Depends,
		PreDepends:   p.PreDepends,
		Conflicts:    p.Conflicts,
		Breaks:       p.Breaks,
		Replaces:     p.Replaces,
		Section:      p.Section,
		Priority:     p.Priority,
		Homepage:     p.Homepage,
	}
}

// Filename derives the standard debian filename as package-version-arch.deb
// based on the data specified in PackageSpec.
func (c Control) debFilename() string {
	return fmt.Sprintf("%s-%s-%s.deb", c.Package, c.Version, c.Architecture)
}

// CalculateChecksums produces the contents of the md5sums file with the
// following format:
//
//	checksum  file1
//	checksum  file2
//
// All files returned by ListFiles() are included
func md5sums(fs vfs.NameSpace) (string, error) {
	var buf bytes.Buffer
	vfs.Walk("/", fs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := fs.Open(path)
		if err != nil {
			return err
		}
		hash := md5.New()
		_, err = io.Copy(hash, file)
		if err != nil {
			return err
		}
		sum := hash.Sum([]byte{})
		hexSum := hex.EncodeToString(sum)
		fmt.Fprintf(&buf, "%v  %s\n", hexSum, path)
		return nil
	})
	return buf.String(), nil
}

func confFiles(fs vfs.NameSpace, patterns ...string) (string, error) {
	var buf bytes.Buffer
	vfs.Walk("/", fs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		for _, pattern := range patterns {
			ok, err := filepath.Match(pattern, path)
			if err != nil {
				return err
			}
			if ok {
				fmt.Fprintln(&buf, path)
				return nil
			}
		}
		return nil
	})
	if _, err := buf.WriteString("\n"); err != nil {
		return "", err

	}
	return buf.String(), nil
}

func vfsTotalFileSize(fs vfs.NameSpace) (int64, error) {
	var size int64
	vfs.Walk("/", fs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	// Convert size from bytes to kilobytes. If there is a remainder, round up.
	if size%1024 > 0 {
		size = size/1024 + 1
	} else {
		size = size / 1024
	}

	return size, nil
}

func (c *Control) BuildArchive(target string, fspec *Files) error {
	fsmap := make(map[string]string)
	{
		sumData, err := md5sums(fspec.Data)
		if err != nil {
			return err
		}
		fsmap["md5sums"] = sumData
	}
	{
		confData, err := confFiles(fspec.Data, "/etc/*/**")
		if err != nil {
			return err
		}
		fsmap["conffiles"] = confData
	}
	{
		size, err := vfsTotalFileSize(fspec.Data)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		err = c.WriteControl(&buf, size)
		if err != nil {
			return err
		}
		fsmap["control"] = buf.String()
	}
	mmap := make(map[string]os.FileMode)
	{
		for _, v := range controlFiles {
			mmap[v] = 0775
		}
		for _, v := range []string{"control", "md5sums", "conffiles"} {
			mmap[v] = 0644
		}
	}
	var file *os.File
	{
		var err error
		file, err = os.Create(target)
		if err != nil {
			return fmt.Errorf("Failed to create control archive %q: %s", target, err)
		}
	}
	defer file.Close()

	// Create a compressed archive stream
	zipwriter := pgzip.NewWriter(file)
	defer zipwriter.Close()
	archive := tar.NewWriter(zipwriter)
	defer archive.Close()

	fs := vfs.NewNameSpace()
	fs.Bind("/", fspec.Control, "/", vfs.BindReplace)
	fs.Bind("/", vfs.ModeMap(vfs.Map(fsmap), mmap), "/", vfs.BindAfter)

	setDefaults := func(h *tar.Header) {
		h.ModTime = time.Now()
		h.Uname = "root"
		h.Gname = "root"
	}

	vfs.Walk("/", fs, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		log.Println("path", path)
		h, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		setDefaults(h)
		filename := strings.TrimPrefix(path, "/")
		if filename == "" {
			return nil
		}
		h.Name = filename
		if err := archive.WriteHeader(h); err != nil {
			return err
		}
		if !fi.IsDir() {
			f, err := fs.Open(path)
			if err != nil {
				return err
			}
			_, err = io.Copy(archive, f)
			f.Close()
			if err != nil {
				return err
			}

		}
		return nil
	})
	return nil
}

func (c Control) withDefaults() Control {
	if c.Priority == "" {
		c.Priority = "optional"
	}
	return c
}

func (c Control) WriteControl(wr io.Writer, installedSize int64) error {
	s := struct {
		Control
		InstalledSize int64
	}{
		Control:       c.withDefaults(),
		InstalledSize: installedSize,
	}
	return controlTemplate.Execute(wr, s)
}

// Validate checks the syntax of various text fields in PackageSpec to verify
// that they conform to the debian package specification. Errors from this call
// should be passed to the user so they can fix errors in their config file.
func (p Control) Validate() error {
	p = p.withDefaults()
	// Verify required fields are specified
	missing := []string{}
	if p.Package == "" {
		missing = append(missing, "package")
	}
	if p.Version == "" {
		missing = append(missing, "version")
	}
	if p.Architecture == "" {
		missing = append(missing, "architecture")
	}
	if p.Maintainer == "" {
		missing = append(missing, "maintainer")
	}
	if p.Description == "" {
		missing = append(missing, "description")
	}
	if len(missing) > 0 {
		return fmt.Errorf("These required fields are missing: %s", strings.Join(missing, ", "))
	}

	hasString := func(items []string, search string) bool {
		for _, item := range items {
			if item == search {
				return true
			}
		}
		return false
	}

	if !hasString(supportedArchitectures, p.Architecture) {
		return fmt.Errorf("Arch %q is not supported; expected one of %s",
			p.Architecture, strings.Join(supportedArchitectures, ", "))
	}
	for _, dep := range p.Depends {
		if !reDepends.MatchString(dep) {
			return fmt.Errorf("Dependency %q is invalid; expected something like 'libc (= 5.1.2)' matching %q", dep, reDepends.String())
		}
	}
	for _, dep := range p.PreDepends {
		if !reDepends.MatchString(dep) {
			return fmt.Errorf("PreDependency %q is invalid; expected something like 'libc (= 5.1.2)' matching %q", dep, reDepends.String())
		}
	}
	for _, replace := range p.Replaces {
		if !reReplacesEtc.MatchString(replace) {
			return fmt.Errorf("Replacement %q is invalid; expected something like 'libc (<< 5.1.2)' matching %q", replace, reReplacesEtc.String())
		}
	}
	for _, conflict := range p.Conflicts {
		if !reReplacesEtc.MatchString(conflict) {
			return fmt.Errorf("Conflict %q is invalid; expected something like 'libc (<< 5.1.2)' matching %q", conflict, reReplacesEtc.String())
		}
	}
	for _, breaks := range p.Breaks {
		if !reReplacesEtc.MatchString(breaks) {
			return fmt.Errorf("Break %q is invalid; expected something like 'libc (<< 5.1.2)' matching %q", breaks, reReplacesEtc.String())
		}
	}
	return nil
}

var (
	reDepends = regexp.MustCompile(
		`^[a-zA-Z0-9.+_-]+( \((>|>=|<|<=|=) ([0-9][0-9a-zA-Z.-]*?)\))?$`)

	reReplacesEtc = regexp.MustCompile(
		`^[a-zA-Z0-9.+_-]+( \(<< ([0-9][0-9a-zA-Z.-]*?)\))?$`)

	controlTemplate = template.
			Must(template.
				New("controlfile").
				Funcs(template.FuncMap{
				"join": func(s []string) string {
					return strings.Join(s, ", ")
				},
			}).
			Parse(`Package: {{ .Package }}
Version: {{ .Version }}
Architecture: {{ .Architecture}}
Maintainer: {{ .Maintainer }}
Installed-Size: {{ .InstalledSize }}
{{- if (len .PreDepends) gt 0 }}
Pre-Depends: {{ join .PreDepends }}
{{- end -}}
{{- if (len .Depends) gt 0 }}
Depends: {{ join .Depends }}
{{- end -}}
{{- if (len .Conflicts) gt 0 }}
Conflicts: {{ join .Conflicts }}
{{- end -}}
{{- if (len .Breaks) gt 0 }}
Breaks: {{ join .Breaks }}
{{- end -}}
{{- if (len .Replaces) gt 0 }}
Replaces: {{ join .Replaces }}
{{- end }}
{{- if (len .Section) gt 0 }}
Section: {{ .Section }}
{{- end }}
{{- if (len .Priority) gt 0 }}
Priority: {{ .Priority }}
{{- end }}
{{- if (len .Homepage) gt 0 }}
Homepage: {{ .Homepage }}
{{- end }}
Description: {{ .Description }}
`))
)

var (
	controlFiles = []string{
		"preinst",
		"postinst",
		"prerm",
		"postrm",
	}

	supportedArchitectures = []string{
		"all", // This is used for non-binary packages
		"amd64",
		"arm64",
		"armel",
		"armhf",
		"i386",
		"mips",
		"mipsel",
		"powerpc",
		"ppc64el",
		"s390x",
	}
)
