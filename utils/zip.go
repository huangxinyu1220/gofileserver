package utils

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func CompressZipFile(w io.Writer, dir string) {
	zw := zip.NewWriter(w)
	defer zw.Close()
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		relPath := filepath.Base(dir) + path[len(dir):]
		absPath := path
		return addFileToZip(zw, relPath, absPath)
	})
}

func statFile(file string) (fi os.FileInfo, reader io.ReadCloser, err error) {
	fi, err = os.Lstat(file)
	if err != nil {
		return
	}
	if fi.Mode() & os.ModeSymlink != 0 {
		target, err1 := os.Readlink(file)
		if err1 != nil {
			err = err1
			return
		}
		reader = ioutil.NopCloser(bytes.NewBufferString(target))
	} else if fi.IsDir() {
		reader = ioutil.NopCloser(bytes.NewBuffer(nil))
	} else {
		reader, err = os.Open(file)
		if err != nil {
			return
		}
	}
	return
}

func addFileToZip(zw *zip.Writer, relPath string, absPath string) error {
	fi, reader, err := statFile(absPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	header, err:= zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		header.Name = relPath + "/"
	} else {
		header.Name = relPath
	}
	header.Method = zip.Deflate
	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, reader)
	if err != nil {
		return err
	}
	return nil
}

func DecompressZipFile(zipFile string, dst string) error {
	zf, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zf.Close()
	for _, f := range zf.File {
		filename := filepath.Join(dst, f.Name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(filename, 0755)
			if err != nil {
				return err
			}
		} else {
			srcFile, err := f.Open()
			if err != nil {
				return err
			}
			defer srcFile.Close()
			dstFile, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer dstFile.Close()
			n, err := io.Copy(dstFile, srcFile)
			if err != nil {
				return err
			}
			log.Println("copy", n, "bytes successfully")
		}
	}
	return nil
}