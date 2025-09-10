package pkg

type RecordData struct {
	Audio      string `json:"audio"`
	Duration   int    `json:"duration"`
	Channels   int    `json:"channels"`
	SampleRate int    `json:"sample_rate"`
	SampleSize int    `json:"sample_size"`
}

type Couple struct {
	SongID     string `json:"song_id"`
	AnchorTime uint32 `json:"anchor_time"`
}
