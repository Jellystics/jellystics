package jellyfin

// ItemsResponse is the generic paged items response from Jellyfin.
type ItemsResponse struct {
	Items            []Item `json:"Items"`
	TotalRecordCount int    `json:"TotalRecordCount"`
	StartIndex       int    `json:"StartIndex"`
}

// Item represents a Jellyfin library item (Movie, Series, Episode, Audio, MusicAlbum, etc.)
type Item struct {
	Id                  string          `json:"Id"`
	Name                string          `json:"Name"`
	ServerId            string          `json:"ServerId"`
	Type                string          `json:"Type"`
	CollectionType      string          `json:"CollectionType"`
	PremiereDate        *string         `json:"PremiereDate"`
	DateCreated         *string         `json:"DateCreated"`
	EndDate             *string         `json:"EndDate"`
	CommunityRating     *float64        `json:"CommunityRating"`
	OfficialRating      *string         `json:"OfficialRating"`
	RunTimeTicks        *int64          `json:"RunTimeTicks"`
	ProductionYear      *int            `json:"ProductionYear"`
	IsFolder            bool            `json:"IsFolder"`
	Status              *string         `json:"Status"`
	Overview            *string         `json:"Overview"`
	Genres              []string        `json:"Genres"`
	ImageTags           ImageTags       `json:"ImageTags"`
	ImageBlurHashes     ImageBlurHashes `json:"ImageBlurHashes"`
	BackdropImageTags   []string        `json:"BackdropImageTags"`
	ParentId            *string         `json:"ParentId"`
	// Series
	SeriesId            *string         `json:"SeriesId"`
	SeriesName          *string         `json:"SeriesName"`
	SeasonId            *string         `json:"SeasonId"`
	SeasonName          *string         `json:"SeasonName"`
	IndexNumber         *int            `json:"IndexNumber"`
	ParentIndexNumber   *int            `json:"ParentIndexNumber"`
	ParentLogoItemId    *string         `json:"ParentLogoItemId"`
	ParentBackdropItemId *string        `json:"ParentBackdropItemId"`
	ParentBackdropImageTags []string    `json:"ParentBackdropImageTags"`
	SeriesPrimaryImageTag   *string     `json:"SeriesPrimaryImageTag"`
	// Music
	AlbumArtist         *string         `json:"AlbumArtist"`
	Album               *string         `json:"Album"`
	AlbumId             *string         `json:"AlbumId"`
	AlbumArtists        []NamedItem     `json:"AlbumArtists"`
	ArtistItems         []NamedItem     `json:"ArtistItems"`
	// Media info
	MediaSources        []MediaSource   `json:"MediaSources"`
	MediaStreams        []MediaStream   `json:"MediaStreams"`
}

// Backdrop returns the first backdrop image tag or empty string.
func (i *Item) Backdrop() *string {
	if len(i.BackdropImageTags) > 0 {
		return &i.BackdropImageTags[0]
	}
	return nil
}

// PrimaryHash returns the blur hash for the primary image.
func (i *Item) PrimaryHash() *string {
	tag, ok := i.ImageTags["Primary"]
	if !ok {
		return nil
	}
	if i.ImageBlurHashes.Primary == nil {
		return nil
	}
	h, ok := i.ImageBlurHashes.Primary[tag]
	if !ok {
		return nil
	}
	return &h
}

// FirstAlbumArtistId returns the Id of the first AlbumArtist, if any.
func (i *Item) FirstAlbumArtistId() *string {
	if len(i.AlbumArtists) > 0 {
		id := i.AlbumArtists[0].Id
		return &id
	}
	return nil
}

// ParentBackdrop returns the first parent backdrop tag.
func (i *Item) ParentBackdrop() *string {
	if len(i.ParentBackdropImageTags) > 0 {
		return &i.ParentBackdropImageTags[0]
	}
	return nil
}

type ImageTags map[string]string

type ImageBlurHashes struct {
	Primary map[string]string `json:"Primary"`
	Thumb   map[string]string `json:"Thumb"`
	Logo    map[string]string `json:"Logo"`
	Banner  map[string]string `json:"Banner"`
}

type NamedItem struct {
	Id   string `json:"Id"`
	Name string `json:"Name"`
}

type MediaSource struct {
	Id       string        `json:"Id"`
	Path     string        `json:"Path"`
	Size     *int64        `json:"Size"`
	Bitrate  *int64        `json:"Bitrate"`
	Container string       `json:"Container"`
	MediaStreams []MediaStream `json:"MediaStreams"`
}

type MediaStream struct {
	Codec         string  `json:"Codec"`
	Type          string  `json:"Type"`
	BitRate       *int64  `json:"BitRate"`
	Height        *int    `json:"Height"`
	Width         *int    `json:"Width"`
	DisplayTitle  *string `json:"DisplayTitle"`
	Language      *string `json:"Language"`
	IsDefault     bool    `json:"IsDefault"`
	Index         int     `json:"Index"`
	IsExternal    bool    `json:"IsExternal"`
	VideoRange     FlexString `json:"VideoRange"`     // SDR | HDR
	VideoRangeType FlexString `json:"VideoRangeType"` // SDR | HDR10 | HLG | DOVI | ...
}

// Library is the Jellyfin virtual folder / library entry.
type Library struct {
	ItemId         string    `json:"ItemId"`
	Id             string    `json:"Id"`
	Name           string    `json:"Name"`
	ServerId       string    `json:"ServerId"`
	IsFolder       bool      `json:"IsFolder"`
	Type           string    `json:"Type"`
	CollectionType string    `json:"CollectionType"`
	ImageTags      ImageTags `json:"ImageTags"`
}

// LibrariesResponse wraps the Items list from /Users/Views.
type LibrariesResponse struct {
	Items []Library `json:"Items"`
}

// User is a Jellyfin user.
type User struct {
	Id                string     `json:"Id"`
	Name              string     `json:"Name"`
	PrimaryImageTag   *string    `json:"PrimaryImageTag"`
	LastLoginDate     *string    `json:"LastLoginDate"`
	LastActivityDate  *string    `json:"LastActivityDate"`
	Policy            UserPolicy `json:"Policy"`
}

type UserPolicy struct {
	IsAdministrator bool `json:"IsAdministrator"`
}

// AuthResponse is returned by /Users/AuthenticateByName.
type AuthResponse struct {
	AccessToken string `json:"AccessToken"`
	User        User   `json:"User"`
}

// SessionInfo represents an active playback session.
type SessionInfo struct {
	Id                   string         `json:"Id"`
	UserId               string         `json:"UserId"`
	UserName             string         `json:"UserName"`
	Client               string         `json:"Client"`
	DeviceName           string         `json:"DeviceName"`
	DeviceId             string         `json:"DeviceId"`
	ApplicationVersion   string         `json:"ApplicationVersion"`
	NowPlayingItem       *SessionItem   `json:"NowPlayingItem"`
	PlayState            *PlayState     `json:"PlayState"`
	TranscodingInfo      *TranscodingInfo `json:"TranscodingInfo"`
	RemoteEndPoint       *string        `json:"RemoteEndPoint"`
	ServerId             *string        `json:"ServerId"`
	IsActive             bool           `json:"IsActive"`
	// LastPlaybackCheckIn is only updated by real client progress reports;
	// LastActivityDate is also bumped by the server-side 1 s position
	// extrapolation. Their gap tells whether a playing client went silent.
	LastPlaybackCheckIn FlexTime `json:"LastPlaybackCheckIn"`
	LastActivityDate    FlexTime `json:"LastActivityDate"`
	LastPausedDate      FlexTime `json:"LastPausedDate"`
	PlaylistItemId      *string  `json:"PlaylistItemId"`
}

type SessionItem struct {
	Id           string        `json:"Id"`
	Name         string        `json:"Name"`
	Type         string        `json:"Type"`
	SeriesName   *string       `json:"SeriesName"`
	SeasonId     *string       `json:"SeasonId"`
	SeriesId     *string       `json:"SeriesId"`
	Container    *string       `json:"Container"`
	MediaStreams  []MediaStream `json:"MediaStreams"`
	// Music fields
	AlbumId      *string       `json:"AlbumId"`
	AlbumArtist  *string       `json:"AlbumArtist"`
	Album        *string       `json:"Album"`
	RunTimeTicks *int64        `json:"RunTimeTicks"`
	ExtraType    *string       `json:"ExtraType"` // ThemeSong, ThemeVideo, Trailer... (not real plays)
}

type PlayState struct {
	PositionTicks *int64  `json:"PositionTicks"`
	IsPaused      bool    `json:"IsPaused"`
	PlayMethod    *string `json:"PlayMethod"`
	// AudioStreamIndex / SubtitleStreamIndex are MediaStream.Index values
	// (subtitle -1 or absent: none). Verified in PlayerStateInfo.cs.
	AudioStreamIndex    *int `json:"AudioStreamIndex"`
	SubtitleStreamIndex *int `json:"SubtitleStreamIndex"`
}

type TranscodingInfo struct {
	VideoCodec    *string `json:"VideoCodec"`
	AudioCodec    *string `json:"AudioCodec"`
	Container     *string `json:"Container"`
	IsVideoDirect bool    `json:"IsVideoDirect"`
	IsAudioDirect bool    `json:"IsAudioDirect"`
	Bitrate       *int64  `json:"Bitrate"`
	Width         *int    `json:"Width"`
	Height        *int    `json:"Height"`
	// TranscodeReasons is a [Flags] enum: array of strings, or comma-separated.
	TranscodeReasons         FlexStringList `json:"TranscodeReasons"`
	HardwareAccelerationType FlexString     `json:"HardwareAccelerationType"`
}

// Plugin is a Jellyfin installed plugin entry.
type Plugin struct {
	Name                  string  `json:"Name"`
	ConfigurationFileName *string `json:"ConfigurationFileName"`
}

// CustomQueryResponse is the response from /user_usage_stats/submit_custom_query.
type CustomQueryResponse struct {
	Results [][]interface{} `json:"results"`
}

// PlaybackReportingRow holds one row from the PlaybackReporting plugin SQLite query.
// Column order: [0]=rowid [1]=DateCreated [2]=UserId [3]=ItemId [4]=ItemType
//               [5]=ItemName [6]=PlaybackMethod [7]=ClientName [8]=DeviceName [9]=PlayDuration
type PlaybackReportingRow = []interface{}

// UserItemData is the per-user playback state of an item.
type UserItemData struct {
	PlaybackPositionTicks int64    `json:"PlaybackPositionTicks"`
	Played                bool     `json:"Played"`
	PlayCount             int      `json:"PlayCount"`
	LastPlayedDate        FlexTime `json:"LastPlayedDate"`
}
