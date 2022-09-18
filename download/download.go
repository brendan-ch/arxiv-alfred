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

	query := os.Args[1]
	downloadFolder := os.Args[2]
	PDFNameTemplate := os.Args[3]

	articleID, err := findArticleID(query)
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
	fileName, err := generateFileName(article, PDFNameTemplate)
	if err != nil {
		log.Fatal(err)
		return
	}
	filePath := filepath.Join(downloadFolder, fileName+".pdf")
	if err != nil {
		log.Fatal(err)
		return
	}

	pdfURL := fmt.Sprintf("https://arxiv.org/pdf/%s.pdf", articleID)
	log.Printf("downloading %s to \"%s\"", pdfURL, filePath)
	err = downloadFile(pdfURL, filePath)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func findArticleID(query string) (string, error) {

	// try to extract the article ID (https://arxiv.org/help/arxiv_identifier) from the query
	patterns := [...]string{
		`\d{4}.\d{4,5}(?:v\d+)?`,               // ID since April 2007
		`[a-z]+(?:-[a-z]+)?\/\d{5,7}(?:v\d+)?`, // ID up to March 2007
	}
	for _, pattern := range patterns {
		r := regexp.MustCompile(pattern)
		if r.MatchString(query) {
			return r.FindString(query), nil
		}
	}
	return "", errors.New("couldn't extract the article ID from the provided query")
}

func generateFileName(article *gofeed.Item, PDFNameTemplate string) (string, error) {

	title := article.Title
	title = strings.Replace(title, "\n ", "", -1)
	title = strings.Replace(title, " :", ":", -1)
	title = strings.Replace(title, ":", " -", -1)

	firstAuthorFullName := article.Authors[0].Name
	firstAuthorLastName := strings.Join(strings.Split(firstAuthorFullName, " ")[1:], " ")

	authorsStringFullName := firstAuthorFullName
	authorsStringLastName := firstAuthorLastName
	for _, author := range article.Authors[1:] {
		authorsStringFullName += ", " + author.Name
		authorsStringFullName += ", " + strings.Join(strings.Split(author.Name, " ")[1:], " ")
	}

	t, err := time.Parse(time.RFC3339, article.Published)
	if err != nil {
		return "", err
	}
	year := fmt.Sprintf("%d", t.Year())
	month := fmt.Sprintf("%02d", t.Month())

	et_al := "et al"
	if len(article.Authors) == 1 {
		et_al = ""
	}

	URLparts := strings.Split(article.GUID, "/")
	ID := URLparts[len(URLparts)-1]

	substitutions := map[string]string{
		"firstauthor_fullname": firstAuthorFullName,
		"firstauthor_lastname": firstAuthorLastName,
		"authors_fullname":     authorsStringFullName,
		"authors_lastname":     authorsStringLastName,
		"year":                 year,
		"month":                month,
		"title":                title,
		"et_al":                et_al,
		"id":                   ID,
	}

	fileName := PDFNameTemplate
	for placeholder, substitution := range substitutions {
		fileName = strings.Replace(fileName, "%"+placeholder+"%", substitution, -1)
	}
	return fileName, nil

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
