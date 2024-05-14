package main

import (
	zip2 "archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:generate go run download.go

const version = "1.0.167"

func main() {
	err := os.RemoveAll("build/protobuf/")
	if err != nil {
		panic("Couldn't delete directory build/protobuf")
	}

	err = os.RemoveAll("build/generated/")
	if err != nil {
		panic("Couldn't delete directory build/protobuf")
	}

	file := fmt.Sprintf("build/protobuf/api-%s.zip", version)

	err = os.MkdirAll("build/protobuf", os.ModePerm)
	if err != nil {
		panic("Couldn't create directory build/protobuf")
	}

	err = os.MkdirAll("build/generated", os.ModePerm)
	if err != nil {
		panic("Couldn't create directory build/protobuf")
	}

	err = downloadFile(file, fmt.Sprintf("https://github.com/SourceForgery/tachikoma/releases/download/%s/tachikoma-frontend-api-proto-%s.zip", version, version))
	if err != nil {
		panic(err)
	}

	unzipFile(file, "build/protobuf/")
}

func unzipFile(zip string, destDir string) {
	archive, err := zip2.OpenReader(zip)
	if err != nil {
		panic(err)
	}
	defer func(archive *zip2.ReadCloser) {
		_ = archive.Close()
	}(archive)

	for _, f := range archive.File {
		filePath := filepath.Join(destDir, f.Name)

		if !strings.HasPrefix(filePath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				panic("Couldn't create directory: " + err.Error())
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}
		defer func(dstFile *os.File) {
			_ = dstFile.Close()
		}(dstFile)

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}
		defer func(fileInArchive io.ReadCloser) {
			_ = fileInArchive.Close()
		}(fileInArchive)

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}
	}
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	_, err = io.Copy(out, resp.Body)
	return err
}
