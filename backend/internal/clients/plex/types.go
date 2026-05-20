package plex

type PlexResponse struct {
	MediaContainer MediaContainer `json:"MediaContainer"`
}

type MediaContainer struct {
	Directory []Directory `json:"Directory"`
	Metadata  []Metadata  `json:"Metadata"`
}

type Directory struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type Metadata struct {
	Media []Media `json:"media"`
}

type Media struct {
	Part []Part `json:"part"`
}

type Part struct {
	ID  int    `json:"id"`
	Key string `json:"key"`
}
