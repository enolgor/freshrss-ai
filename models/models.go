package models

type Feed struct {
	Name    string  `json:"name"`
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Content string `json:"content"`
}

type UnreadFeeds struct {
	Date  string `json:"date"`
	Feeds []Feed `json:"feeds"`
}
