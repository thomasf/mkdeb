package deb

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cbednarski/mkdeb/deb/tar"
	"github.com/klauspost/pgzip"
	"github.com/thomasf/vfs"
)

func createDataArchive(target string, fs vfs.NameSpace) error {
	file, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("Failed to create data archive %q: %s", target, err)
	}
	defer file.Close()

	// Create a compressed archive stream
	zipwriter := pgzip.NewWriter(file)
	defer zipwriter.Close()
	archive := tar.NewWriter(zipwriter)
	defer archive.Close()

	setDefaults := func(h *tar.Header) {
		h.Uid = 0
		h.Gid = 0
		h.Uname = "root"
		h.Gname = "root"
	}

	err = vfs.Walk("/", fs, func(path string, fi os.FileInfo, err error) error {
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
	if err != nil {
		return err
	}

	return nil
}
