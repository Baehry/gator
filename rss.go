package main

import (
	"net/http"
	"encoding/xml"
	"io"
	"html"
	"context"
	"time"
	"fmt"
	"database/sql"
	"github.com/google/uuid"
	"github.com/Baehry/gator/internal/database"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var rssFeed RSSFeed
	if err := xml.Unmarshal(data, &rssFeed); err != nil {
		return nil, err
	}
	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
	rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)
	for i, item := range rssFeed.Channel.Item {
		rssFeed.Channel.Item[i].Title = html.UnescapeString(item.Title)
		rssFeed.Channel.Item[i].Description = html.UnescapeString(item.Description)
	}
	return &rssFeed, nil
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}
	if err := s.db.MarkFeedFetched(context.Background(), feed.ID); err != nil {
		return err
	}
	fmt.Printf("%s:\n", feed.Name)
	rssFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}
	for _, item := range rssFeed.Channel.Item {
		_, err = s.db.GetPost(context.Background(), item.Link)
		if err != sql.ErrNoRows && err != nil {
			return err
		} else if err != sql.ErrNoRows {
			continue
		}
		publishDate, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700",item.PubDate)
		if err != nil {
			return err
		}
		params := database.CreatePostParams{
			ID: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Title: html.UnescapeString(item.Title),
			Url: item.Link,
			Description: html.UnescapeString(item.Description),
			PublishedAt: publishDate,
			FeedID: feed.ID,
		}
		if _, err = s.db.CreatePost(context.Background(), params); err != nil {
			return err
		}
	}
	return nil
}