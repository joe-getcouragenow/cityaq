package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Get correct on for each OS. Assume x86 ISA chip.

// ProtoURLPrefix is the download Url
const ProtoURLPrefix = "https://github.com/protocolbuffers/protobuf/releases/download/v3.11.4/"

// ProtocZipFilenameMAC is the Mac filename
const ProtocZipFilenameMAC = "protoc-3.11.4-osx-x86_64.zip"

// ProtocZipFilenameLIN is the Linux filename
const ProtocZipFilenameLIN = "protoc-3.11.4-linux-x86_64.zip"

// ProtocZipFilenameWIN is the Windows filename
const ProtocZipFilenameWIN = "protoc-3.11.4-win64.zip"

func main() {
	// Download protoc from: https://github.com/protocolbuffers/protobuf/releases/tag/v3.11.4

	// ProtocZipFilename final
	var protocZipFilename = ""

	if runtime.GOOS == "windows" {
		fmt.Println("You are running on Windows")
		protocZipFilename = ProtocZipFilenameWIN
	}

	if runtime.GOOS == "darwin" {
		fmt.Println("You are running on Mac")
		protocZipFilename = ProtocZipFilenameMAC
	}

	if runtime.GOOS == "linux" {
		fmt.Println("You are running on Linux")
		protocZipFilename = ProtocZipFilenameLIN
	}

	var downloadURL = ProtoURLPrefix + protocZipFilename

	fmt.Println("-- Downloading --")
	fmt.Println("Downloading source Url: " + downloadURL)
	fmt.Println("Downloading output fileName: " + protocZipFilename)
	err := downloadFile(protocZipFilename, downloadURL)
	if err != nil {
		log.Fatal(err)
	}

	var unzipOutFilePath = "lib-protoc"
	fmt.Println("-- Unzipping --")
	fmt.Println("Unzipping source fileName: " + protocZipFilename)
	fmt.Println("Unzipping outout filePath: " + unzipOutFilePath)
	files, err := unzip(protocZipFilename, unzipOutFilePath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Unzipped:\n" + strings.Join(files, "\n"))

	//curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v3.11.4/$(Protoc-zip-filename)
	//mkdir -p ./lib-protoc
	//unzip -a ./$(Protoc-zip-filename) -d ./lib-protoc

}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(filepath string, url string) error {

	// check if file already downloaded !
	if fileExists(filepath) {
		fmt.Println("file exists: " + filepath)
		return nil
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		/*
			if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
				return filenames, fmt.Errorf("%s: illegal file path", fpath)
			}
		*/

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
