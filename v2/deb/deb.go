package deb

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"bytes"

	"github.com/laher/argo/ar"
)

// Deb .
type Deb struct {
	*Control
	*Files
}

// Build creates a .deb file in the target directory. The name is defived from
// Filename() so you can find it with:
//
//	path.Join(target, PackageSpec.Filename())

// debWriter .
type debWriter struct {
	*ar.Writer
}

func (d *debWriter) writeDebianBinary() error {
	r := bytes.NewBufferString("2.0\n")
	h := ar.Header{
		ModTime: time.Now(),
		Uid:     0,
		Gid:     0,
		Mode:    0600,
		Size:    int64(r.Len()),
		Name:    "debian-binary",
	}
	if err := d.WriteHeader(&h); err != nil {
		return err
	}
	if _, err := io.Copy(d, r); err != nil {
		return err
	}
	return nil
}

func (d *debWriter) writeFile(name string, r io.Reader, size int64) error {
	h := ar.Header{
		ModTime: time.Now(),
		Uid:     0,
		Gid:     0,
		Mode:    0600,
		Size:    size,
		Name:    name,
	}
	if err := d.WriteHeader(&h); err != nil {
		return err
	}
	n, err := io.Copy(d, r)
	if err != nil {
		return err
	}
	if n != size {
		return fmt.Errorf(
			"bytes written does not match with provided size: %v : %v",
			n, size)
	}
	return nil
}

// func createDeb2(
// 	target string,
// 	ws string, // temp pathj
// 	dataArchiveCh, controlArchiveCh chan string,
// 	errCh chan error) error {

// 	// 1. Create binary package (tar.gz format)
// 	// 2. Create control file package (tar.gz format)
// 	// 3. Create .deb / package (ar archive format)

// 	err := os.MkdirAll(target, 0755)
// 	if err != nil {
// 		return fmt.Errorf("Unable to create target directory %q: %s", target, err)
// 	}

// 	file, err := os.Create(target)
// 	if err != nil {
// 		return fmt.Errorf("Failed to create build target: %s", err)
// 	}
// 	defer file.Close()

// 	archive := debWriter(ar.NewWriter(file))
// 	archiveCreationTime := time.Now()

// 	select {
// 	case controlFile := <-controlArchiveCh:
// 		// Copy the control file archive into ar (.deb)
// 		if err := writeFileToAr(archive, baseHeader, controlFile); err != nil {
// 			return err
// 		}
// 	case err := <-errCh:
// 		return err
// 	}

// 	select {
// 	case dataFile := <-dataArchiveCh:
// 		// Copy the data archive into the ar (.deb)
// 		if err := writeFileToAr(archive, baseHeader, dataFile); err != nil {
// 			return err
// 		}
// 	case err := <-errCh:
// 		return err
// 	}

// 	if err := archive.Close(); err != nil {
// 		return err
// 	}
// 	if err := file.Close(); err != nil {
// 		return err
// 	}
// 	return nil
// }

func createDeb(target string, fspec *Files, c Control) error {
	err := c.Validate()
	if err != nil {
		log.Println(err)
		// return err
	}
	ws, err := ioutil.TempDir("", "mkdeb")
	if err != nil {
		return fmt.Errorf("Could not create build workspace: %v", err)
	}
	defer func() {
		err := os.RemoveAll(ws) // clean up
		if err != nil {
			log.Printf("Error cleaning up build workspace '%v': %v", ws, err)
		}
	}()

	err = os.MkdirAll(target, 0755)
	if err != nil {
		return fmt.Errorf("Unable to create target directory %q: %s", target, err)
	}

	file, err := os.Create(path.Join(target, c.debFilename()))
	if err != nil {
		return fmt.Errorf("Failed to create build target: %s", err)
	}
	defer file.Close()
	archive := debWriter{ar.NewWriter(file)}
	defer archive.Close()
	{
		if err := archive.writeDebianBinary(); err != nil {
			return fmt.Errorf("Failed to write debian-binary: %v", err)
		}
	}
	{
		controlFile := filepath.Join(ws, "control.tar.gz")
		if err := c.BuildArchive(controlFile, fspec); err != nil {
			return fmt.Errorf("Failed to compress control files: %v", err)
		}
		stat, err := os.Stat(controlFile)
		if err != nil {
			return err
		}
		f, err := os.Open(controlFile)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := archive.writeFile("control.tar.gz", f, stat.Size()); err != nil {
			return err
		}
		f.Close()
	}
	{
		dataFile := filepath.Join(ws, "data.tar.gz")
		if err := createDataArchive(dataFile, fspec.Data); err != nil {
			return fmt.Errorf("Failed to compress data files: %s", err)
		}
		stat, err := os.Stat(dataFile)
		if err != nil {
			return err
		}
		f, err := os.Open(dataFile)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := archive.writeFile("data.tar.gz", f, stat.Size()); err != nil {
			return err
		}
		f.Close()

	}

	if err := archive.Close(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return nil
}
