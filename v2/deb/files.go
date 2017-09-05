package deb

import (
	"os"
	"path/filepath"

	deb1 "github.com/cbednarski/mkdeb/deb"
	"github.com/pkg/errors"
	"github.com/thomasf/vfs"
)

// Files .
type Files struct {
	Data      vfs.NameSpace // data archive
	Control   vfs.NameSpace // control archive
	Conffiles []string      // path.Match Glob option, defaults to "/etc/*/**", use a single dash "-" to disable default value.
}

type FilesFunc func(fs *Files) error

func AutoPath(path string) FilesFunc {
	return func(fs *Files) error {
		{
			stat, err := os.Stat(path)
			if err != nil {
				return errors.Wrapf(err, "auto path failed to add: %s", path)
			}
			if !stat.IsDir() {
				return errors.Errorf("auto path: path is not a directory: %s", path)
			}
		}
		{
			var excludePaths []string
			for _, cf := range controlFiles {
				excludePaths = append(excludePaths, "/"+cf)
			}
			fs.Data.Bind(
				"/", vfs.Exclude(vfs.OS(path), excludePaths...),
				"/", vfs.BindBefore)
		}
		controlFileMap := make(map[string]string)
		for _, cf := range controlFiles {
			cfp := filepath.Join(path, cf)
			if _, err := os.Stat(cfp); err == nil {
				controlFileMap[cf] = cfp
			}
		}
		if len(controlFileMap) > 0 {
			fs.Control.Bind(
				"/", vfs.FileMap(controlFileMap),
				"/", vfs.BindBefore)
		}
		return nil
	}
}

func fromPackageSpec(p *deb1.PackageSpec) *Files {
	dataFs := vfs.NewNameSpace()
	controlFs := vfs.NewNameSpace()
	controlFileMap := make(map[string]string)
	controlFs.Bind("/", vfs.Map(controlFileMap), "/", vfs.BindBefore)
	if p.AutoPath != "" {
		var excls []string
		for _, cf := range controlFiles {
			excls = append(excls, "/"+cf)
		}
		dataFs.Bind("/", vfs.Exclude(vfs.OS(p.AutoPath), excls...), "/", vfs.BindAfter)
		cFileMap := make(map[string]string)
		for _, cf := range controlFiles {
			cfp := filepath.Join(p.AutoPath, cf)
			if _, err := os.Stat(cfp); err == nil {
				cFileMap[cf] = cfp
			}
		}
		if len(cFileMap) > 0 {
			controlFs.Bind("/", vfs.FileMap(cFileMap), "/", vfs.BindAfter)
		}
	}
	if p.Files != nil && len(p.Files) > 0 {
		fmap := make(map[string]string)
		for k, v := range p.Files {
			// todo: validate that stuff exists...
			fmap[v] = k
		}
		dataFs.Bind("/", vfs.FileMap(fmap), "/", vfs.BindBefore)
	}
	fs := &Files{
		Data:    dataFs,
		Control: controlFs,
	}
	return fs
	// data:       dataFs,
	// control:    controlFs,
	// controlMap: controlFileMap,

}
