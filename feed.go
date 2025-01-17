package main

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
        "encoding/json"

	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

type Feed struct {
	Name string
	URL  string
}

var wg sync.WaitGroup
var isAllUpdate bool

const newArticleDirectory = "new"
const maxFileNameLength = 255

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for !utf8.ValidString(s[:n]) {
		n--
	}
	return s[:n]
}

func InitialiseNewArticleDirectory() {
	DeleteFeedFiles(newArticleDirectory)
	os.MkdirAll(Config.FeedDirectory+"/"+newArticleDirectory, 0755)
}

func DeleteFeedFiles(name string) {
	os.RemoveAll(Config.FeedDirectory + "/" + name)
}

type FeedJSONFile struct {
	Link string `json:"link"`
	PubDate string `json:"pubDate"`
}

func UpdateFeed(name string, deleteFiles bool) {
	log.Info("Updating feed '" + name + "'")
	fp := gofeed.NewParser()
	downloadCount := 0
	skipCount := 0
	feed, err := fp.ParseURL(Config.Feeds[slices.IndexFunc(Config.Feeds, func(f Feed) bool { return f.Name == name })].URL)
	if err != nil {
		log.Error("Failed to fetch the feed '" + name + "'")
		if isAllUpdate {
			wg.Done()
		}
		return
	}
	if deleteFiles {
		DeleteFeedFiles(name)
	}
	os.MkdirAll(Config.FeedDirectory+"/"+name, 0777)
	for _, item := range feed.Items {
		articlePath := Config.FeedDirectory + "/" + name + "/" + truncateString(strings.ReplaceAll(item.Title, "/", ""), maxFileNameLength)
		if _, err := os.Stat(articlePath); err == nil {
			log.Debug("Article " + articlePath + " already exists - skipping download")
			skipCount++
			continue
		}
		file, err := os.Create(articlePath)
		if err != nil {
			log.Error("Failed to create a file for article titled '" + item.Title + "'")
			continue
		}
		defer file.Close()

                jsonData := &FeedJSONFile{
                  Link: item.Link,
                  PubDate: item.Published,
                }

                // marshal
                jsonBytes, err := json.Marshal(jsonData)

                if err != nil {
                  log.Error("Failed to create json for article titled '" + item.Title + "'")
                  continue
                }

                _, err = file.WriteString(string(jsonBytes))

		if err != nil {
			log.Error("Failed to write content to a file for article titled '" + item.Title + "'")
			continue
		}
		downloadCount++
		newLinkPath := Config.FeedDirectory + "/" + newArticleDirectory + "/" + truncateString(strings.ReplaceAll(item.Title, "/", ""), maxFileNameLength)
		err = os.Symlink(articlePath, newLinkPath)
		if err != nil {
			log.Error("Could not create symlink for newly downloaded article " + articlePath)
		}
	}
	log.Info(strconv.Itoa(downloadCount) + " articles fetched from feed '" + name + "' (" + strconv.Itoa(skipCount) + " already seen, " + strconv.Itoa(len(feed.Items)) + " total in feed)")
	if isAllUpdate {
		wg.Done()
	}
}

func UpdateAllFeeds(deleteFiles bool) {
	isAllUpdate = true
	for _, feed := range Config.Feeds {
		wg.Add(1)
		go UpdateFeed(feed.Name, deleteFiles)
	}
	wg.Wait()
}
