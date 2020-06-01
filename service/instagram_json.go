package service

type InstagramAdditionalData struct {
	GraphQL *InstagramGraphQL `json:"graphql"`
}

type InstagramGraphQL struct {
	ShortcodeMedia *InstagramShortcodeMedia `json:"shortcode_media"`
}

type InstagramShortcodeMedia struct {
	DisplayURL            string                `json:"display_url"`
	IsVideo               bool                  `json:"is_video,omitempty"`
	EdgeSideCarToChildren *InstagramEdgeSideCar `json:"edge_sidecar_to_children,omitempty"`
}

type InstagramEdgeSideCar struct {
	Edges []*InstagramEdge `json:"edges"`
}

type InstagramEdge struct {
	Node *InstagramNode `json:"node"`
}

type InstagramNode struct {
	DisplayURL string `json:"display_url"`
	IsVideo    bool   `json:"is_video,omitempty"`
}
