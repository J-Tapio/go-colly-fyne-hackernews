package main

import (
	"strings"
	"log"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/gocolly/colly"
	"github.com/pkg/browser"
)

type newsStory struct {
	URL string
	Img string
	Title string
	Date string
	Author string
	Caption string
}


// Scraped news
var newsStories []newsStory
// News elements
var latestNews []fyne.CanvasObject

var hackerNewsApp fyne.App
var appWindow fyne.Window
var hnImg fyne.Resource
var scrapeError bool

func init() {
	// Initialize & Configure Fyne App
	hackerNewsApp = app.New()
	hackerNewsApp.Settings().SetTheme(&myTheme{})
	// Configure app window
	appWindow = hackerNewsApp.NewWindow("The Hacker News - Latest")
	hnIconResource, err := fyne.LoadResourceFromPath("./hn-icon.png")
	if err != nil {
		log.Printf("Error with loading HackerNews icon: %s", err)
	} else {
		hnImg = hnIconResource
		appWindow.SetIcon(hnImg)
	}
	appWindow.Resize(fyne.NewSize(400, 800))
}


func main() {
	
	go func() {
		scrapeError = false
		// Resource to scrape
		hackerNewsURL := "https://thehackernews.com"
		// Channels
		log.Println("Fetching latest stories")
		fromHN := make(chan newsStory, 7)
		toList := make(chan newsStory, 7)
	
		go scrapeNews(hackerNewsURL, fromHN)
		go outputToStories(toList)

		hnOpen := true
		for hnOpen {
			story, open := <-fromHN
			if open {
				toList <-story
			} else {
				hnOpen = false
			}
		}

		if !scrapeError {
			log.Println("Done fetching new stories")
			createNewsFeedView()
		} else {
			log.Println("Couldn't fetch the latest news")
			createErrorView()
		}

		time.Sleep(30 * time.Minute)
	}()

	appWindow.ShowAndRun()
	tidyUp()
}

func tidyUp() {
	log.Println("Exited app.")
}

func createErrorView() {
	// Show error and time when trying for update again
	nextUpdateTry := time.Now().Add(time.Minute * 30).Format(time.Kitchen)
	errorTitleEl := canvas.NewText("Unfortunately something went wrong with fetching latest news.", color.White)
	errorTitleEl.Alignment = fyne.TextAlignCenter
	errorTitleEl.TextStyle.Bold = true
	tryAgainText := "Trying again: " + nextUpdateTry
	tryAgainTextEl := canvas.NewText(tryAgainText, color.White)
	tryAgainTextEl.Alignment = fyne.TextAlignCenter
	tryAgainTextEl.TextStyle.Bold = true

	hnImageElement := canvas.NewImageFromResource(hnImg)
	hnImageElement.FillMode = canvas.ImageFillContain
	hnImageElement.SetMinSize(fyne.NewSize(50, 50))

	errorContent := container.NewVBox(layout.NewSpacer(),hnImageElement, errorTitleEl, tryAgainTextEl, layout.NewSpacer())

	appWindow.SetContent(errorContent)
	appWindow.Canvas().Refresh(errorContent)
}

func createNewsFeedView() {
	// Update news & refresh view
	latestNews = createNewsStoryNodes(newsStories)
	content := container.NewVScroll(
		container.New(layout.NewVBoxLayout(), latestNews...),
	)
	appWindow.SetContent(content)
	appWindow.Canvas().Refresh(content)
}

func outputToStories(c <- chan newsStory) {
	for {
		story := <- c
		newsStories = append(newsStories, story)
	}
}

func scrapeNews(url string, c chan<- newsStory) {
	collector := colly.NewCollector()

	collector.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
	})

	collector.OnError(func(_ *colly.Response, err error) {
		//? Maybe use log.fatal -> SIGINT(1) etc?
    log.Println("Something went wrong:", err)
		scrapeError = true
		close(c)
	})

	collector.OnResponse(func(r *colly.Response) {
		log.Println("Visited", r.Request.URL)
	})

	collector.OnHTML(".body-post", func(e *colly.HTMLElement) {
		story := newsStory{}
		// Nodes, which need further sub-string selection
		storyDateNode := e.ChildText(".item-label")
		storyAuthorNode := e.ChildText(".item-label span")
		// News story
		story.URL = e.ChildAttr(".story-link", "href")
		story.Img = e.ChildAttr(".img-ratio img", "data-src")
		story.Title = e.ChildText(".home-title")
		// Remove the icon and return text
		story.Date = storyDateNode[3:16]
		// Remove the icon and return text
		story.Author = storyAuthorNode[3:]
		story.Caption = e.ChildText(".home-desc")

		c <-story
	})

	collector.OnScraped(func(r *colly.Response) {
		log.Println("Finished with news scraping", r.Request.URL)
		close(c)
	})

	collector.Visit(url)
}


func createNewsStoryNodes(news []newsStory) []fyne.CanvasObject {
	newsNodes := []fyne.CanvasObject{}
	nextUpdateTime := time.Now().Add(time.Minute * 30).Format(time.Kitchen)

	// Info element to show at top before news
	// Title
	overallTitleEl := canvas.NewText("Latest news from The Hacker News", color.White)
	overallTitleEl.Alignment = fyne.TextAlignCenter
	overallTitleEl.TextStyle.Bold = true
	// Next update
	nextUpdateText := "Next update: " + nextUpdateTime
	nextUpdateEl := canvas.NewText(nextUpdateText, color.White)
	nextUpdateEl.Alignment = fyne.TextAlignCenter
	nextUpdateEl.TextStyle.Bold = true

	overallInfo := container.NewGridWithRows(5, layout.NewSpacer(), overallTitleEl, nextUpdateEl, layout.NewSpacer())

	newsNodes = append(newsNodes, overallInfo)

	// Iterate through news and create fyne element
	for _, story := range newsStories {

		// Story image
		img, _ := fyne.LoadResourceFromURLString(story.Img)
		storyImg := canvas.NewImageFromResource(img)
		storyImg.FillMode = canvas.ImageFillContain
		storyImg.Translucency = 0.8

		// Story title
		title := canvas.NewText(story.Title, color.White)
		title.Alignment = fyne.TextAlignCenter
		title.TextStyle.Bold = true

		// Story date & author
		dateAndAuthorText := strings.Join([]string{story.Author, "-", story.Date}, " ")
		dateAndAuthor := canvas.NewText(dateAndAuthorText, color.White)
		dateAndAuthor.Alignment = fyne.TextAlignCenter

		// Story link button
		storyURL := story.URL
		storyLinkBtn := widget.NewButtonWithIcon(
			"Read the article", 
			hnImg, 
			func() {
			browser.OpenURL(storyURL)
		})
		storyLinkBtn.Importance = widget.MediumImportance

		// Create card-like layout and insert elements
		storyInfo := container.NewGridWithRows(5, layout.NewSpacer(), title, dateAndAuthor, layout.NewSpacer(), storyLinkBtn)
		storyNode := container.NewPadded(storyImg, storyInfo)

		newsNodes = append(newsNodes, storyNode)
	}

	return newsNodes
}