package feed

import (
	"errors"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/nbd-wtf/go-nostr"
	"github.com/piraces/rsslay/pkg/converter"
)

const (
	KindLongFormTextContent = 30023
)

type ItemToEventConverter interface {
	Convert(pubkey string, item *gofeed.Item, feed *gofeed.Feed, defaultCreatedAt time.Time, originalUrl string) nostr.Event
}

type ConverterSelector struct {
	noteConverter     ItemToEventConverter
	longFormConverter ItemToEventConverter
}

func NewConverterSelector(noteConverter ItemToEventConverter, longFormConverter ItemToEventConverter) *ConverterSelector {
	return &ConverterSelector{noteConverter: noteConverter, longFormConverter: longFormConverter}
}

func (s *ConverterSelector) Select(feed *gofeed.Feed) ItemToEventConverter {
	if strings.Contains(feed.Link, "substack.com") {
		return s.longFormConverter
	}
	return s.noteConverter
}

type NoteConverter struct {
	maxContentLength int
}

func NewNoteConverter(maxContentLength int) (*NoteConverter, error) {
	if maxContentLength <= 0 {
		return nil, errors.New("max content length must be a positive number")
	}
	return &NoteConverter{maxContentLength: maxContentLength}, nil
}

func (s *NoteConverter) Convert(pubkey string, item *gofeed.Item, feed *gofeed.Feed, defaultCreatedAt time.Time, originalUrl string) nostr.Event {
	content := buildContent(item, feed, originalUrl, s.maxContentLength, converter.GetNoteConverterRules())

	createdAt := defaultCreatedAt
	if item.UpdatedParsed != nil {
		createdAt = *item.UpdatedParsed
	}
	if item.PublishedParsed != nil {
		createdAt = *item.PublishedParsed
	}

	evt := nostr.Event{
		PubKey:    pubkey,
		CreatedAt: nostr.Timestamp(createdAt.Unix()),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{},
		Content:   content,
	}
	evt.ID = string(evt.Serialize())

	return evt
}

type LongFormConverter struct {
}

func NewLongFormConverter() *LongFormConverter {
	return &LongFormConverter{}
}

func (l *LongFormConverter) Convert(pubkey string, item *gofeed.Item, feed *gofeed.Feed, defaultCreatedAt time.Time, originalUrl string) nostr.Event {
	content := buildContent(item, feed, originalUrl, 0, converter.GetLongFormConverterRules())

	createdAt := defaultCreatedAt
	if item.UpdatedParsed != nil {
		createdAt = *item.UpdatedParsed
	}
	if item.PublishedParsed != nil {
		createdAt = *item.PublishedParsed
	}

	tags := nostr.Tags{
		[]string{"published_at", strconv.FormatInt(createdAt.Unix(), 10)},
	}

	if item.GUID != "" {
		tags = append(tags, []string{"d", item.GUID})
	}

	if item.Title != "" {
		tags = append(tags, []string{"title", item.Title})
	}

	evt := nostr.Event{
		PubKey:    pubkey,
		CreatedAt: nostr.Timestamp(createdAt.Unix()),
		Kind:      KindLongFormTextContent,
		Tags:      tags,
		Content:   content,
	}
	evt.ID = string(evt.Serialize())

	return evt
}

func buildContent(item *gofeed.Item, feed *gofeed.Feed, originalUrl string, maxContentLength int, converterRules []md.Rule) string {
	content := ""
	if item.Title != "" {
		content = "**" + item.Title + "**"
	}

	itemDescription := htmlToMarkdown(item.Description, converterRules)
	itemContent := htmlToMarkdown(item.Content, converterRules)

	if maxContentLength == 0 && len(itemContent) != 0 {
		content += "\n\n" + itemContent
	} else {
		if !strings.EqualFold(item.Title, itemDescription) && !strings.Contains(feed.Link, "stacker.news") {
			content += "\n\n" + itemDescription
		}
	}

	shouldUpgradeLinkSchema := false

	// Handle Nitter special cases (duplicates and http schema)
	if strings.Contains(feed.Description, "Twitter feed") {
		content = ""
		shouldUpgradeLinkSchema = true

		if strings.HasPrefix(originalUrl, "https://") {
			itemDescription = strings.ReplaceAll(itemDescription, "http://", "https://")
		}

		if strings.Contains(item.Title, "RT by @") {
			if len(item.DublinCoreExt.Creator) > 0 {
				content = "**" + "RT " + item.DublinCoreExt.Creator[0] + ":**\n\n"
			}
		} else if strings.Contains(item.Title, "R to @") {
			fields := strings.Fields(item.Title)
			if len(fields) >= 2 {
				replyingToHandle := fields[2]
				content = "**" + "Response to " + replyingToHandle + "**\n\n"
			}
		}
		content += itemDescription
	}

	content = html.UnescapeString(content)
	if maxContentLength > 0 && len(content) > maxContentLength {
		content = content[0:(maxContentLength-1)] + "…"
	}

	if shouldUpgradeLinkSchema {
		item.Link = strings.ReplaceAll(item.Link, "http://", "https://")
	}

	// Handle comments
	if item.Custom != nil {
		if comments, ok := item.Custom["comments"]; ok {
			content += fmt.Sprintf("\n\nComments: %s", comments)
		}
	}

	content += "\n\n" + item.Link

	return strings.ToValidUTF8(content, "")
}

func htmlToMarkdown(s string, converterRules []md.Rule) string {
	mdConverter := md.NewConverter("", true, nil)
	mdConverter.AddRules(converterRules...)

	convertedS, err := mdConverter.ConvertString(s)
	if err != nil {
		log.Printf("[WARN] failure to convert to markdown (defaulting to plain text): %v", err)
		p := bluemonday.StripTagsPolicy()
		convertedS = p.Sanitize(s)
	}

	return convertedS
}
