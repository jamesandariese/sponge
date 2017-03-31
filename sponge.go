package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

/*
   TODO:
   - parameterize items to get
   - parameterize output location
   - general cleanup
   - other sources
       - wash post
       - wsj
       - reddit
            - python
            - programming
            - sysadmin
            - others?
       - techcrunch?
       - economist
   - filter Hacker News if no url
*/
var itemsToFetch = 10

// this makes a pointer to a string that gets filled in your main function.
var outputLocation = flag.String("out", "/tmp/sponge_out.txt", "Output file")

type Formatted struct {
	Body string
}

/*
   HACKER NEWS
*/

type HackerNewsItem struct {
	Title string `json:"title"`
	Url   string `json:"url"`
}

func (h HackerNewsItem) getFormatted() Formatted {
	return Formatted{
		Body: fmt.Sprintf("Title: %s\nUrl: %s", h.Title, h.Url)}
}

func getHackerNews() []Formatted, err {
	hackerNewsListUrl := "https://hacker-news.firebaseio.com/v0/topstories.json"
	hackerNewsItemUrl := "https://hacker-news.firebaseio.com/v0/item/%d.json"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(hackerNewsListUrl)
	if err != nil {
		// https://play.golang.org/p/S_zMsozkyc
		return nil, err
	}
	defer resp.Body.Close()

	var hnl []int
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&hnl)

	// take only top items (returns 500 initially)
	var hnTopItems []int
	hnTopItems = append(hnTopItems, ...hnl[:itemsToFetch])

	var hnTopItemsDetails []Formatted

	collectorChan := make(chan Formatted)
	
	var wg sync.WaitGroup
	wg.Add(len(hnTopItems))

	for _, id := range hnTopItems {
		// id := id
		//
		// I prefer doing ^^^ but you are doing a perfectly valid pattern by passing the id in as its same name.
		// I like the closure method rather than the parameter passing method because it doesn't require knowledge
		// of the beginning of the function at the end unless it's actually necessary.
		go func(id int) {
			defer wg.Done()

			resp, err := client.Get(fmt.Sprintf(hackerNewsItemUrl, id))
			if err != nil {
				fmt.Println("error!", err)
				return
			}
			defer resp.Body.Close()

			item := HackerNewsItem{}
			decoder := json.NewDecoder(resp.Body)
			if dec_err := decoder.Decode(&item); dec_err != nil {
				print("error!", err)
			} else {
				collectorChan <- item.getFormatted()
			}
		}(id)
	}
	
	go func() {
		// this is in the goroutine because the range over the channel needs to block
		// exit of the outer function.
		// This goroutine with its waitgroup and close will guarantee that the for loop
		// runs entirely and then exits when the waitgroup is finished.
		// the wg.Add must happen *before* this goroutine is spawned but this goroutine
		// can exist anywhere after the wg.Add, including *just* after the wg.Add if you
		// want to keep the control code together.  I'd usually recommend that but I wanted
		// to keep the diff somewhat locally relevant.
		wg.Wait()
		close(collectorChan)
	}()
	
	for item := range collectorChan {
		hnTopItemsDetails = append(hnTopItemsDetails, item)
	}

	return hnTopItemsDetails
}

/*
   REDDIT GOLANG
*/

type RedditItem struct {
	Title string `json:"title"`
	Url   string `json:"url"`
}

func (r RedditItem) getFormatted() Formatted {
	return Formatted{
		Body: fmt.Sprintf("Title: %s\nUrl: %s", r.Title, r.Url)}
}

type RedditList struct {
	Data struct {
		Children []struct {
			Data RedditItem `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func getRedditGolang() []Formatted {
	redditUsernameEnvName := "REDDIT_USERNAME"

	redditUsername := os.Getenv(redditUsernameEnvName)
	if redditUsername == "" {
		fmt.Printf("Env var %s not found!\n", redditUsernameEnvName)
		return make([]Formatted, 0)
	}
	userAgent := fmt.Sprintf("golang Sponge:0.0.1 (by /u/%s)", redditUsername)

	golangListUrl := fmt.Sprintf("https://www.reddit.com/r/golang/top.json?raw_json=1&t=day&limit=%d", itemsToFetch)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", golangListUrl, nil)
	req.Header.Set("User-Agent", userAgent) // required or reddit API will return 429 code
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return make([]Formatted, 0)
	}
	if resp.StatusCode != 200 {
		fmt.Println("Non 200 response", resp.StatusCode)
		return make([]Formatted, 0)
	}
	defer resp.Body.Close()

	redditList := RedditList{}
	decoder := json.NewDecoder(resp.Body)
	decodeErr := decoder.Decode(&redditList)
	if decodeErr != nil {
		fmt.Println(decodeErr)
		return make([]Formatted, 0)
	}

	redditItems := make([]Formatted, itemsToFetch)
	for i, item := range redditList.Data.Children {
		redditItems[i] = item.Data.getFormatted()
	}

	return redditItems
}

/*
   NEW YORK TIMES
*/

type NytItem struct {
	Title string `json:"title"`
	Url   string `json:"url"`
}

func (n NytItem) getFormatted() Formatted {
	return Formatted{
		Body: fmt.Sprintf("Title: %s\nUrl: %s", n.Title, n.Url)}
}

type NytList struct {
	Results []NytItem `json:"results"`
}

func getNyt() []Formatted {
	apiKeyName := "NYT_API_KEY"

	nytApiKey := os.Getenv(apiKeyName)
	if nytApiKey == "" {
		fmt.Printf("Env var %s not found!\n", apiKeyName)
		return make([]Formatted, 0)
	}

	nytUrl := fmt.Sprintf("https://api.nytimes.com/svc/topstories/v2/home.json?api-key=%s", nytApiKey)

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(nytUrl)

	if err != nil {
		fmt.Println(err)
		return make([]Formatted, 0)
	}
	defer resp.Body.Close()

	nytList := NytList{}
	decoder := json.NewDecoder(resp.Body)
	decodeErr := decoder.Decode(&nytList)
	if decodeErr != nil {
		fmt.Println(err)
		return make([]Formatted, 0)
	}

	nytItems := make([]Formatted, itemsToFetch)
	for i := 0; i < itemsToFetch; i++ {
		item := nytList.Results[i]
		nytItems[i] = item.getFormatted()
	}

	return nytItems
}

/*
   File creation
*/

func writeSection(f *os.File, sectionName string, items []Formatted) {
	heading := fmt.Sprintf("\n\n=====================================\n"+
		"%s\n=====================================\n\n", sectionName)
	f.WriteString(heading)

	for _, item := range items {
		f.WriteString(item.Body)
		f.WriteString("\n\n")
	}

	fmt.Printf("wrote %d %s items\n", len(items), sectionName)
}

func main() {
	flag.Parse()

	redditItems := getRedditGolang()
	hackerNewsItems := getHackerNews()
	nytItems := getNyt()

	f, err := os.Create(*outputLocation)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	writeSection(f, "New York Times", nytItems)
	writeSection(f, "Hacker News", hackerNewsItems)
	writeSection(f, "Reddit Golang", redditItems)

	fmt.Printf("Done writing output to %s\n", *outputLocation)
}
