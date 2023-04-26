package converter

import (
	"fmt"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

func GetNoteConverterRules() []md.Rule {
	return []md.Rule{
		{
			Filter: []string{"h1", "h2", "h3", "h4", "h5", "h6"},
			Replacement: func(content string, selection *goquery.Selection, opt *md.Options) *string {
				content = strings.TrimSpace(content)
				return md.String(content)
			},
		},
		{
			Filter: []string{"img"},
			AdvancedReplacement: func(content string, selec *goquery.Selection, opt *md.Options) (md.AdvancedResult, bool) {
				src := selec.AttrOr("src", "")
				src = strings.TrimSpace(src)
				if src == "" {
					return md.AdvancedResult{
						Markdown: "",
					}, false
				}

				src = opt.GetAbsoluteURL(selec, src, "")

				text := fmt.Sprintf("\n%s\n", src)
				return md.AdvancedResult{
					Markdown: text,
				}, false
			},
		},
		convererRuleA,
	}
}

func GetLongFormConverterRules() []md.Rule {
	return []md.Rule{
		{
			Filter: []string{"h1", "h2", "h3", "h4", "h5", "h6"},
			Replacement: func(content string, selection *goquery.Selection, opt *md.Options) *string {
				content = strings.TrimSpace(content)
				return md.String(content)
			},
		},
		//{
		//	Filter: []string{"img"},
		//	AdvancedReplacement: func(content string, selec *goquery.Selection, opt *md.Options) (md.AdvancedResult, bool) {
		//		src := selec.AttrOr("src", "")
		//		src = strings.TrimSpace(src)
		//		if src == "" {
		//			return md.AdvancedResult{
		//				Markdown: "",
		//			}, false
		//		}
		//
		//		src = opt.GetAbsoluteURL(selec, src, "")
		//
		//		text := fmt.Sprintf("![]%s", src)
		//		return md.AdvancedResult{
		//			Markdown: text,
		//		}, false
		//	},
		//},
		convererRuleA,
	}
}

var convererRuleA = md.Rule{
	Filter: []string{"a"},
	AdvancedReplacement: func(content string, selec *goquery.Selection, opt *md.Options) (md.AdvancedResult, bool) {
		// if there is no href, no link is used. So just return the content inside the link
		href, ok := selec.Attr("href")
		if !ok || strings.TrimSpace(href) == "" || strings.TrimSpace(href) == "#" {
			return md.AdvancedResult{
				Markdown: content,
			}, false
		}

		href = opt.GetAbsoluteURL(selec, href, "")

		// having multiline content inside a link is a bit tricky
		content = md.EscapeMultiLine(content)

		// if there is no link content (for example because it contains an svg)
		// the 'title' or 'aria-label' attribute is used instead.
		if strings.TrimSpace(content) == "" {
			content = selec.AttrOr("title", selec.AttrOr("aria-label", ""))
		}

		// a link without text won't de displayed anyway
		if content == "" {
			return md.AdvancedResult{
				Markdown: "",
			}, false
		}

		replacement := fmt.Sprintf("%s (%s)", content, href)

		return md.AdvancedResult{
			Markdown: replacement,
		}, false
	},
}
