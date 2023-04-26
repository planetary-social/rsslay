package feed

import (
	"testing"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
)

const feedWithComments = `<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
<channel>
<title>Stacker News</title>
<link>https://stacker.news</link>
<description>Like Hacker News, but we pay you Bitcoin.</description>
<language>en</language>
<lastBuildDate>Sat, 18 Feb 2023 12:35:17 GMT</lastBuildDate>
<atom:link href="https://stacker.news/rss" rel="self" type="application/rss+xml"/>
<item>
<guid>https://stacker.news/items/138518</guid>
<title>What is your favourite Linux distribution, and why?</title>
<link>https://stacker.news/items/138518</link>
<comments>https://stacker.news/items/138518</comments>
<description>
<![CDATA[ <a href="https://stacker.news/items/138518">Comments</a> ]]>
</description>
<pubDate>Fri, 17 Feb 2023 18:29:20 GMT</pubDate>
</item>
</channel>
</rss>`

const feedWithoutComments = `<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
<channel>
<title>Stacker News</title>
<link>https://stacker.news</link>
<description>Like Hacker News, but we pay you Bitcoin.</description>
<language>en</language>
<lastBuildDate>Sat, 18 Feb 2023 12:35:17 GMT</lastBuildDate>
<atom:link href="https://stacker.news/rss" rel="self" type="application/rss+xml"/>
<item>
<guid>https://stacker.news/items/138518</guid>
<title>What is your favourite Linux distribution, and why?</title>
<link>https://stacker.news/items/138518</link>
<description>
<![CDATA[ <a href="https://stacker.news/items/138518">Comments</a> ]]>
</description>
<pubDate>Fri, 17 Feb 2023 18:29:20 GMT</pubDate>
</item>
</channel>
</rss>`

func TestCustomTranslator_TranslateWithComments(t *testing.T) {
	fp := gofeed.NewParser()
	fp.RSSTranslator = NewCustomTranslator()
	feed, _ := fp.ParseString(feedWithComments)
	item := feed.Items[0]
	assert.NotNil(t, item.Custom)
	assert.NotNil(t, item.Custom["comments"])
	assert.Equal(t, "https://stacker.news/items/138518", item.Custom["comments"])
}

func TestCustomTranslator_TranslateWithoutComments(t *testing.T) {
	fp := gofeed.NewParser()
	fp.RSSTranslator = NewCustomTranslator()
	feed, _ := fp.ParseString(feedWithoutComments)
	item := feed.Items[0]
	assert.Nil(t, item.Custom)
}
