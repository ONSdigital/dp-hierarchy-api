package models

import "fmt"

const codelistFormat = "%s/code-list/%s/code"

var CodelistURL string

// Response models a node in the hierarchy
type Response struct {
	ID          string          `json:"-"`
	Label       string          `json:"label"`
	Children    []*Element      `json:"children,omitempty"`
	Links       map[string]Link `json:"links,omitempty"`
	HasData     bool            `json:"-"`
	Breadcrumbs []*Element      `json:"breadcrumbs,omitempty"`
	Hierarchy   Hierarchy       `json:"-"`
}

// Element is a item in a list within a Response
type Element struct {
	ID           string          `json:"-"`
	Label        string          `json:"label"`
	NoOfChildren int64           `json:"no_of_children,omitempty"`
	Links        map[string]Link `json:"links,omitempty"`
	HasData      bool            `json:"has_data"`
}

type Link struct {
	ID   string `json:"id,omitempty"`
	HRef string `json:"href,omitempty"`
}

// AddLinks adds links (self, codelist and populates children links)
func (r *Response) AddLinks(hierarchy *Hierarchy, isRoot bool) {
	if r.Links == nil {
		r.Links = make(map[string]Link)
	}
	if isRoot {
		r.Links["self"] = *GetLink(hierarchy.URL, "")
	} else {
		r.Links["self"] = *GetLink(hierarchy.URL, r.ID)
	}
	r.Links["codelist"] = *GetLink(fmt.Sprintf(codelistFormat, CodelistURL, hierarchy.CodelistId), r.ID)
	for _, child := range r.Children {
		child.AddLinks(hierarchy)
	}
}

// AddLinks adds self and codelist links for Elements
func (r *Element) AddLinks(hierarchy *Hierarchy) {
	if r.Links == nil {
		r.Links = make(map[string]Link)
	}
	r.Links["self"] = *GetLink(hierarchy.URL, r.ID)
	r.Links["codelist"] = *GetLink(fmt.Sprintf(codelistFormat, CodelistURL, hierarchy.CodelistId), r.ID)
}

// GetLink returns a Link{id,href} object for the given url/id (or just url if id is empty)
func GetLink(baseURL string, id string) *Link {
	if id == "" {
		return &Link{HRef: baseURL}
	} else {
		return &Link{HRef: baseURL + "/" + id, ID: id}
	}
}
