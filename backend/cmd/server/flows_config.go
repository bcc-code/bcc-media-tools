package main

// vidispineConfig satisfies vsapi.Config.
type vidispineConfig struct{ baseURL, username, password string }

func (c vidispineConfig) BaseURL() string  { return c.baseURL }
func (c vidispineConfig) Username() string { return c.username }
func (c vidispineConfig) Password() string { return c.password }

// cantemoConfig satisfies cantemo.Config.
type cantemoConfig struct{ url, token string }

func (c cantemoConfig) URL() string   { return c.url }
func (c cantemoConfig) Token() string { return c.token }
