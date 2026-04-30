package config

var ROCKY_WEBSITE = &WebsiteConfig{
	Name:         "Rocky",
	URL:          "https://lexore.ca/rocky/api",
	IsPrimary:    true,
	IsPredefined: true,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/upload",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}

var BALLCHASING_WEBSITE = &WebsiteConfig{
	Name:         "Ballchasing",
	URL:          "https://ballchasing.com/api",
	IsPrimary:    false,
	IsPredefined: true,
	URIParams:    map[string]string{"vibilitity": "public"},
	NeedToken:    true,
	Token:        "",
	SendPing:     true,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/v2/upload",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}

var LOCAL_WEBSITE = &WebsiteConfig{
	Name:         "Localhost",
	URL:          "http://localhost:3000",
	IsPrimary:    true,
	IsPredefined: true,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/upload",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}
