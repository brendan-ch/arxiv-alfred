package main

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/mmcdole/gofeed"
)

var wf *aw.Workflow

func main() {
	wf.Run(run)
}

func init() {
	wf = aw.New()
}

func run() {

	args := wf.Args()
	if len(args) > 1 {
		wf.FatalError(fmt.Errorf("unexpected number of arguments"))
		return
	}
	query := args[0]

	items, err := searchItems(query)
	if err != nil {
		wf.FatalError(err)
		return
	}

	prepareItems(items)
}

func searchItems(query string) ([]*gofeed.Item, error) {

	// try to extract the article ID (https://arxiv.org/help/arxiv_identifier) from the query and retrieve the article ...
	patterns := [...]string{
		`\d{4}.\d{4,5}(?:v\d+)?`,               // ID since April 2007
		`[a-z]+(?:-[a-z]+)?\/\d{5,7}(?:v\d+)?`, // ID up to March 2007
	}
	for _, pattern := range patterns {
		r := regexp.MustCompile(pattern)
		if r.MatchString(query) {
			articleID := r.FindString(query)
			log.Printf("Matched pattern, article ID: %s", articleID)
			request := fmt.Sprintf("http://export.arxiv.org/api/query?id_list=%s", articleID)
			return fetchResults(request)
		}
	}

	// ... otherwise search for articles using the provided query
	// preprocess query (this was found to yield the best / most reliable search results)
	re, err := regexp.Compile(`[^0-9a-zA-Z]`)
	if err != nil {
		log.Fatal(err)
	}
	query = re.ReplaceAllString(query, " ")
	query = strings.Replace(query, "  ", " ", -1)

	request := fmt.Sprintf("http://export.arxiv.org/api/query?search_query=%s&sortBy=relevance&max_results=9", url.QueryEscape(query))
	log.Printf("API call: %s", request)
	return fetchResults(request)
}

func fetchResults(request string) ([]*gofeed.Item, error) {
	fp := gofeed.NewParser()
	res, error := fp.ParseURL(request)
	if error != nil {
		return nil, error
	}
	return res.Items, nil
}

func prepareItems(items []*gofeed.Item) {
	if len(items) > 0 {
		for _, item := range items {
			addItem(item)
		}
	} else {
		wf.NewItem("No Results").Valid(false)
	}
	wf.SendFeedback()
}

func addItem(item *gofeed.Item) {

	authorString := item.Authors[0].Name
	for _, author := range item.Authors[1:] {
		authorString += ", " + author.Name
	}

	t, _ := time.Parse(time.RFC3339, item.Published)
	date := fmt.Sprintf("%d-%02d", t.Year(), t.Month())

	title := item.Title
	title = strings.Replace(title, "\n ", "", -1)
	title = strings.Replace(title, " :", ":", -1)

	resultItem := wf.NewItem(title)
	resultItem.Subtitle(fmt.Sprintf("%s %s", date, authorString))
	resultItem.Valid(true)
	resultItem.Arg(item.GUID)

	pdfURL := strings.Replace(item.Link, "abs", "pdf", -1) + ".pdf"
	resultItem.Cmd().Arg(pdfURL)
	resultItem.NewModifier("cmd", "shift").Arg(pdfURL)

}
