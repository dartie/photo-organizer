package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hellflame/argparse"
	timefmt "github.com/itchyny/timefmt-go"
	"github.com/rwcarlsen/goexif/exif"
)

var Version = "1.0.0"
var LINE_UP = "\033[1A"
var LINE_CLEAR = "\x1b[2K"

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

func displayProgress(n int, tot int) {
	percentage := (n * 100) / tot

	fmt.Print(LINE_UP)
	fmt.Print(LINE_CLEAR)
	fmt.Printf("%d%% (%d/%d)\n", percentage, n, tot)
}

func getFilePrefix(filepath string) string {
	imgFile, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err.Error())
	}

	metaData, err := exif.Decode(imgFile)
	if err != nil {
		log.Fatal(err.Error())
	}

	tm, _ := metaData.DateTime()
	myTm := strings.Trim(strings.Split(fmt.Sprintf("%s", tm), "+")[0], " ")

	myTmTime, _ := timefmt.Parse(myTm, "%Y-%m-%d %H:%M:%S")
	dateFormat := "%Y_%m_%d-%H_%M_%S" // "%Y-%m-%d_%H:%M:%S"
	myTmFormatted := timefmt.Format(myTmTime, dateFormat)
	filePrefix := strings.ReplaceAll(myTmFormatted, ":", "_")

	return filePrefix + "--"
}

func main() {
	fmt.Println()
	parser := argparse.NewParser("Photo Organizer  "+Version, `Photo Organizer`, nil)

	/* ---- List of arguments -------------------------------------------------- */
	// customRcfFilePath for user-rcf
	folder := parser.String("", "folder", &argparse.Option{
		Positional: true,
		Validate: func(arg string) error {
			if _, e := os.Stat(arg); e != nil {
				return fmt.Errorf("folder '%s' does not exist", arg)
			}
			return nil
		},
	})

	outputFolder := parser.String("", "output-folder", &argparse.Option{
		Positional: true,
		Validate: func(arg string) error {
			if _, e := os.Stat(arg); e != nil {
				os.MkdirAll(arg, os.ModePerm)
			}
			return nil
		},
	})

	/* ---- Parse -------------------------------------------------------------- */
	if e := parser.Parse(nil); e != nil {
		fmt.Println(e.Error())
		return
	}

	files, err := ioutil.ReadDir(*folder)
	if err != nil {
		log.Fatal(err)
	}

	for n, file := range files {
		displayProgress(n+1, len(files))
		fullPathfile := filepath.Join(*folder, file.Name())

		filePrefix := getFilePrefix(fullPathfile)
		newFilename := fmt.Sprintf("%s%s", filePrefix, file.Name())
		newfullPathfile := filepath.Join(*outputFolder, newFilename)

		// copy the file with the new name
		CopyFile(fullPathfile, newfullPathfile)

	}
}
