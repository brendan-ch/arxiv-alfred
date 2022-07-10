package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

func main() {

	articleURL := os.Args[1]
	downloadFolder := os.Args[2]
	format := os.Args[3]
	articleID, err := findArticleID(articleURL)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("found article with ID %s", articleID)

	fp := gofeed.NewParser()
	request := fmt.Sprintf("http://export.arxiv.org/api/query?id_list=%s", articleID)
	res, err := fp.ParseURL(request)
	if err != nil {
		log.Fatal(err)
		return
	}

	if len(res.Items) != 1 {
		log.Fatal(errors.New("more or less than one article retrieved"))
		return
	}

	article := res.Items[0]
	fileName, err := generateFilename(article, downloadFolder, format)
	if err != nil {
		log.Fatal(err)
		return
	}

	pdfURL := fmt.Sprintf("https://arxiv.org/pdf/%s.pdf", articleID)
	log.Printf("downloading %s to \"%s\"", pdfURL, fileName)
	downloadFile(pdfURL, fileName)
}

func findArticleID(articleURL string) (string, error) {

	// try to extract article ID (https://arxiv.org/help/arxiv_identifier) from query
	patterns := [...]string{
		`\d{4}.\d{4,5}(?:v\d+)?`,               // ID since April 2007
		`[a-z]+(?:-[a-z]+)?\/\d{5,7}(?:v\d+)?`, // ID up to March 2007
	}
	for _, pattern := range patterns {
		r := regexp.MustCompile(pattern)
		if r.MatchString(articleURL) {
			return r.FindString(articleURL), nil
		}
	}
	return "", errors.New("article not found")
}

func generateFilename(article *gofeed.Item, downloadFolder string, format string) (string, error) {

	title := article.Title
	title = strings.Replace(title, "\n ", "", -1)
	title = strings.Replace(title, " :", ":", -1)
	title = strings.Replace(title, ":", " -", -1)

	firstAuthorFullName := article.Authors[0].Name
	firstAuthorLastName := strings.Join(strings.Split(firstAuthorFullName, " ")[1:], " ")

	authorsStringFullName := firstAuthorFullName
	for _, author := range article.Authors[1:] {
		authorsStringFullName += ", " + author.Name
	}
	authorsStringLastName := firstAuthorLastName
	for _, author := range article.Authors[1:] {
		authorsStringFullName += ", " + strings.Join(strings.Split(author.Name, " ")[1:], " ")
	}

	t, _ := time.Parse(time.RFC3339, article.Published)
	year := fmt.Sprintf("%d", t.Year())
	month := fmt.Sprintf("%02d", t.Month())

	et_al := "et al"
	if len(article.Authors) == 1 {
		et_al = ""
	}

	fileName := format
	fileName = strings.Replace(fileName, "%firstauthor_fullname%", firstAuthorFullName, -1)
	fileName = strings.Replace(fileName, "%firstauthor_lastname%", firstAuthorLastName, -1)
	fileName = strings.Replace(fileName, "%authors_fullname%", authorsStringFullName, -1)
	fileName = strings.Replace(fileName, "%authors_lastname%", authorsStringLastName, -1)
	fileName = strings.Replace(fileName, "%year%", year, -1)
	fileName = strings.Replace(fileName, "%month%", month, -1)
	fileName = strings.Replace(fileName, "%title%", title, -1)
	fileName = strings.Replace(fileName, "%et_al%", et_al, -1)

	dirname, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	filePath := filepath.Join(dirname, downloadFolder, fileName+".pdf")
	return filePath, nil

}

func downloadFile(URL, fileName string) error {

	response, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("response code: %d", response.StatusCode)
	}

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}
