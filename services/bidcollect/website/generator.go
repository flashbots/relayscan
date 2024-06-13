package website

//
// Quick and dirty website generator
//

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
)

func BuildProdWebsite(log *logrus.Entry, outDir string, upload bool) {
	log.Infof("Creating build server in %s", outDir)
	err := os.MkdirAll(outDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	dir := "ethereum/mainnet/"

	// Setup minifier
	minifier := minify.New()
	minifier.AddFunc("text/html", html.Minify)
	minifier.AddFunc("text/css", css.Minify)

	// Load month folders from S3
	log.Infof("Getting folders from S3 for %s ...", dir)
	months, err := getFoldersFromS3(dir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Months:", months)

	// build root page
	log.Infof("Building root page ...")
	rootPageData := HTMLData{ //nolint:exhaustruct
		Title:            "",
		Path:             "/index.html",
		EthMainnetMonths: months,
	}

	tpl, err := ParseIndexTemplate()
	if err != nil {
		log.Fatal(err)
	}

	buf := new(bytes.Buffer)
	err = tpl.ExecuteTemplate(buf, "base", rootPageData)
	if err != nil {
		log.Fatal(err)
	}

	// minify
	mBytes, err := minifier.Bytes("text/html", buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	// write to file
	fn := filepath.Join(outDir, "index.html")
	log.Infof("Writing to %s ...", fn)
	err = os.WriteFile(fn, mBytes, 0o0600)
	if err != nil {
		log.Fatal(err)
	}

	toUpload := []struct{ from, to string }{
		{fn, "/"},
	}

	// build files pages
	for _, month := range months {
		dir := "ethereum/mainnet/" + month + "/"
		log.Infof("Getting files from S3 for %s ...", dir)
		files, err := getFilesFromS3(dir)
		if err != nil {
			log.Fatal(err)
		}

		rootPageData := HTMLData{ //nolint:exhaustruct
			Title: month,
			Path:  fmt.Sprintf("ethereum/mainnet/%s/index.html", month),

			CurrentNetwork: "Ethereum Mainnet",
			CurrentMonth:   month,
			Files:          files,
		}

		tpl, err := ParseFilesTemplate()
		if err != nil {
			log.Fatal(err)
		}

		buf := new(bytes.Buffer)
		err = tpl.ExecuteTemplate(buf, "base", rootPageData)
		if err != nil {
			log.Fatal(err)
		}

		// minify
		mBytes, err := minifier.Bytes("text/html", buf.Bytes())
		if err != nil {
			log.Fatal(err)
		}

		// write to file
		_outDir := filepath.Join(outDir, dir)
		err = os.MkdirAll(_outDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		fn := filepath.Join(_outDir, "index.html")
		log.Infof("Writing to %s ...", fn)
		err = os.WriteFile(fn, mBytes, 0o0600)
		if err != nil {
			log.Fatal(err)
		}

		toUpload = append(toUpload, struct{ from, to string }{fn, "/" + dir})
	}

	if upload {
		log.Info("Uploading to S3 ...")
		// for _, file := range toUpload {
		// 	fmt.Printf("- %s -> %s\n", file.from, file.to)
		// }

		for _, file := range toUpload {
			app := "./scripts/bidcollect/s3/upload-file-to-r2.sh"
			cmd := exec.Command(app, file.from, file.to) //nolint:gosec
			stdout, err := cmd.Output()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(stdout))
		}
	}
}

func getFoldersFromS3(dir string) ([]string, error) {
	folders := []string{}

	app := "./scripts/bidcollect/s3/get-folders.sh"
	cmd := exec.Command(app, dir)
	stdout, err := cmd.Output()
	if err != nil {
		return folders, err
	}

	// Print the output
	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		if line != "" && strings.HasPrefix(line, "20") {
			folders = append(folders, strings.TrimSuffix(line, "/"))
		}
	}
	return folders, nil
}

func getFilesFromS3(month string) ([]FileEntry, error) {
	files := []FileEntry{}

	app := "./scripts/bidcollect/s3/get-files.sh"
	cmd := exec.Command(app, month)
	stdout, err := cmd.Output()
	if err != nil {
		return files, err
	}

	space := regexp.MustCompile(`\s+`)
	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		if line != "" {
			line = space.ReplaceAllString(line, " ")
			parts := strings.Split(line, " ")

			// parts[2] is the size
			size, err := strconv.ParseUint(parts[2], 10, 64)
			if err != nil {
				return files, err
			}

			filename := parts[3]

			if filename == "index.html" {
				continue
			} else if strings.HasSuffix(filename, ".csv.gz") {
				continue
			}

			files = append(files, FileEntry{
				Filename: filename,
				Size:     size,
				Modified: parts[1] + " " + parts[0],
			})
		}
	}
	return files, nil
}
