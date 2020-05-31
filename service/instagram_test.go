package service

import (
	"testing"
)

func TestTitle(t *testing.T) {
	var (
		snippet = []byte(`<head>
			<title>Hello
more</title></head>`)
		expected = "Hello\nmore"
	)

	title, err := getDocTitle(snippet)
	if err != nil {
		t.Errorf("parse: %s", err)
	}

	if title != expected {
		t.Errorf("expected: %s actual: %s", expected, title)
	}
}

func TestAdditionalData(t *testing.T) {
	t.Run("single video", func(t *testing.T) {
		var (
			snippet = []byte(`</script><script type="text/javascript">window.__additionalDataLoaded('/p/CA1lPepDJXO/',{"graphql":{"shortcode_media":
				{"is_video":true,"display_url":"https://vurl"}}});</script><`)
			expectedDisplayURL = "https://vurl"
		)

		ad, err := parseAdditionalData(snippet)
		if err != nil {
			t.Errorf("parse: %s", err)
		}

		if !ad.GraphQL.ShortcodeMedia.IsVideo {
			t.Errorf("not video")
		}

		if ad.GraphQL.ShortcodeMedia.DisplayURL != expectedDisplayURL {
			t.Errorf("display url expected: %s actual: %s", expectedDisplayURL, ad.GraphQL.ShortcodeMedia.DisplayURL)
		}
	})

	t.Run("multiple images", func(t *testing.T) {
		var (
			snippet = []byte(`</script><script type="text/javascript">window.__additionalDataLoaded('/p/CA1lPepDJXO/',{"graphql":{"shortcode_media":
				{"is_video":false,"display_url":"https://vurl",
					"foo": "bar",
					"edge_sidecar_to_children":{"edges":[
						{"node":{"display_url":"//1"}},
						{"node":{"display_url":"//2"}},
						{"node":{"display_url":"//3"}}
					]}
				}}});</script><`)
		)

		ad, err := parseAdditionalData(snippet)
		if err != nil {
			t.Errorf("parse: %s", err)
		}

		if ad.GraphQL.ShortcodeMedia.IsVideo {
			t.Errorf("is video")
		}

		if len(ad.GraphQL.ShortcodeMedia.EdgeSideCarToChildren.Edges) != 3 {
			t.Errorf("expected 3 edges, got %d", len(ad.GraphQL.ShortcodeMedia.EdgeSideCarToChildren.Edges))
		}

		if ad.GraphQL.ShortcodeMedia.EdgeSideCarToChildren.Edges[1].Node.DisplayURL != "//2" {
			t.Errorf("unexpected second display url: %s", ad.GraphQL.ShortcodeMedia.EdgeSideCarToChildren.Edges[1].Node.DisplayURL)
		}
	})
}
