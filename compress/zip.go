package gw_compress

import (
	"archive/zip"
	"io"
	"os"
)

func ZipFiles(filename string, files []string, withFilePath bool) error {

	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {
		if err = AddFilePathToZip(zipWriter, file, withFilePath); err != nil {
			return err
		}
	}
	return nil
}

func AddFileToZip(zipWriter *zip.Writer, fileToZip *os.File, withFilePath bool) error {

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	var header *zip.FileHeader
	if withFilePath {
		header, err = zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
	} else {
		header = &zip.FileHeader{}
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	header.Name = fileToZip.Name()

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}

func AddFilePathToZip(zipWriter *zip.Writer, filePath string, withFilePath bool) error {

	fileToZip, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	var header *zip.FileHeader
	if withFilePath {
		header, err = zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
	} else {
		header = &zip.FileHeader{}
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}
