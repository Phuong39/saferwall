// Copyright 2019 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package utils

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v6"
	"github.com/yeka/zip"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GetFileSize returns file size in bytes
func GetFileSize(FilePath string) int64 {
	f, err := os.Stat(FilePath)
	if err != nil {
		log.Fatal(err)
	}
	size := f.Size()
	return size
}

// GetCurrentTime as Time object in UTC
func GetCurrentTime() time.Time {
	return time.Now().UTC()
}

// SliceContainsString returns if slice contains substring
func SliceContainsString(a string, list []string) bool {
	for _, b := range list {
		if strings.Contains(b, a) {
			return true
		}
	}
	return false
}

// StringInSlice returns whether or not a string exists in a slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// UniqueSlice delete duplicate strings from an array of strings
func UniqueSlice(slice []string) []string {
	cleaned := []string{}

	for _, value := range slice {
		if !StringInSlice(value, cleaned) {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

// ExecCommand runs cmd on file
func ExecCommand(name string, args ...string) (string, error) {

	// Create a new context and add a timeout to it
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel() // The cancel should be deferred so resources are cleaned up

	// Create the command with our context
	cmd := exec.CommandContext(ctx, name, args...)

	// We use CombinedOutput() instead of Output()
	// which returns standard output and standard error.
	out, err := cmd.CombinedOutput()

	// We want to check the context error to see if the timeout was executed.
	// The error returned by cmd.Output() will be OS specific based on what
	// happens when a process is killed.
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Println("Command timed out")
		return string(out), err
	}

	// If there's no context error, we know the command completed (or errored).
	return string(out), err
}

// StartCommand will execute a command in background
func StartCommand(name string, args ...string) error {

	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

// Getwd returns the directory where the process is running from
func Getwd() (string, error) {
	return filepath.Abs(filepath.Dir(os.Args[0]))
}

// ReadAll reads the entire file into memory
func ReadAll(filePath string) ([]byte, error) {
	// Start by getting a file descriptor over the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Get the file size to know how much we need to allocate
	fileinfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	filesize := fileinfo.Size()
	buffer := make([]byte, filesize)

	// Read the whole binary
	_, err = file.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer, nil

}

// WalkAllFilesInDir returns list of files in directory
func WalkAllFilesInDir(dir string) ([]string, error) {

	fileList := []string{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		// check if it is a regular file (not dir)
		if info.Mode().IsRegular() {
			fileList = append(fileList, path)
		}
		return nil
	})

	return fileList, err
}

// WriteBytesFile write Bytes to a File.
func WriteBytesFile(filename string, r io.Reader) (int, error) {

	// Open a new file for writing only
	file, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Read for the reader bytes to file
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, err
	}

	// Write bytes to disk
	bytesWritten, err := file.Write(b)
	if err != nil {
		return 0, err
	}

	return bytesWritten, nil
}

// ChownFileUsername executes chown username:username filename
func ChownFileUsername(filename, username string) error {
	group, err := user.Lookup(username)
	if err != nil {
		return err
	}
	uid, _ := strconv.Atoi(group.Uid)
	gid, _ := strconv.Atoi(group.Gid)

	err = os.Chown(filename, uid, gid)
	return err
}

// IsDirectory returns true if path is a directory
func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// GetRootProjectDir retrieves `saferwall` root src project directory.
func GetRootProjectDir() string {
	gopath := os.Getenv("GOPATH")
	sfwRootDir := path.Join(gopath, "src", "github.com", "saferwall", "saferwall")
	return sfwRootDir
}

// CreateFile will create an empty file if file
func CreateFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// path does not exist
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		file.Close()
	}

	return nil
}

// Download downloads an object from a bucket.
func Download(client *minio.Client, bucketName, filePath, objectName string) ([]byte, error) {
	err := client.FGetObject(bucketName, objectName, filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	data, err := ReadAll(filePath)
	return data, err
}

// ZipEncrypt compresses binary data to zip using a password.
func ZipEncrypt(filename string, password string, contents io.Reader) (string, error) {
	zipFilepath := filename + ".zip"
	fzip, err := os.Create(zipFilepath)
	if err != nil {
		return "", err
	}
	zipw := zip.NewWriter(fzip)
	defer zipw.Close()

	w, err := zipw.Encrypt(filename, password, zip.AES256Encryption)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", err
	}
	_, err = io.Copy(w, contents)
	if err != nil {
		return "", err
	}
	zipw.Flush()
	return zipFilepath, nil
}

// ZipDecrypt compresses binary data to zip using a password.
func ZipDecrypt(zipFilepath string, password string) error {
	r, err := zip.OpenReader(zipFilepath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.IsEncrypted() {
			f.SetPassword(password)
		}

		r, err := f.Open()
		if err != nil {
			return err
		}

		buf, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		defer r.Close()

		fmt.Printf("Size of %v: %v byte(s)\n", f.Name, len(buf))
	}

	return nil
}
