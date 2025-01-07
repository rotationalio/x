package api

import "strings"

//===========================================================================
// Top Level Requests and Responses
//===========================================================================

// Reply contains standard fields that are used for generic API responses and errors.
type Reply struct {
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
	ErrorDetail ErrorDetail `json:"errors,omitempty"`
}

// Returned on status requests.
type StatusReply struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime,omitempty"`
	Version string `json:"version,omitempty"`
}

// PageQuery manages paginated list requests.
type PageQuery struct {
	PageSize      int    `json:"page_size,omitempty" url:"page_size,omitempty" form:"page_size"`
	NextPageToken string `json:"next_page_token,omitempty" url:"next_page_token,omitempty" form:"next_page_token"`
	PrevPageToken string `json:"prev_page_token,omitempty" url:"prev_page_token,omitempty" form:"prev_page_token"`
}

// Used to submit search queries to the backend.
type SearchQuery struct {
	Query string `json:"query,omitempty" url:"query,omitempty" form:"query"`
	Limit int    `json:"limit,omitempty" url:"limit,omitempty" form:"limit"`
}

func (q *SearchQuery) Validate() (err error) {
	q.Query = strings.TrimSpace(q.Query)
	if q.Query == "" {
		err = ValidationError(err, MissingField("query"))
	}

	if q.Limit < 0 {
		err = ValidationError(err, IncorrectField("limit", "limit cannot be less than zero"))
	}

	if q.Limit > 50 {
		err = ValidationError(err, IncorrectField("limit", "maximum number of search results that can be returned is 50"))
	}

	return err
}
