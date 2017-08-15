package response

import (
	"strings"

	"github.com/ONSdigital/dp-hierarchy-api/stubs"
	"github.com/ONSdigital/go-ns/log"
)

//Response ...
type Response struct {
	ID        string      `json:"id"`
	LabelCode string      `json:"label_code"`
	Label     string      `json:"label"`
	Children  []*Elements `json:"children,omitempty"`
	Parent    *Elements   `json:"parent,omitempty"`
	URL       string      `json:"url"`
}

//Elements ...
type Elements struct {
	ID           string `json:"id"`
	LabelCode    string `json:"label_code"`
	Label        string `json:"label"`
	NoOfChildren int    `json:"number_of_children,omitempty"`
	CodeListURL  string `json:"code_list_url,omitempty"`
	URL          string `json:"url,omitempty"`
}

func addElements(el []*stubs.Output, label string) []*Elements {
	var list []*Elements
	for _, v := range el {

		//remove wrongful parent elements
		if len(label) != 0 && (len(strings.Split(label, ".")) < len(strings.Split(v.LabelCode, "."))) {
			continue
		}

		e := &Elements{
			ID:          v.ID,
			Label:       v.Label,
			LabelCode:   v.LabelCode,
			CodeListURL: "/code-lists/e44de4c4-d39e-4e2f-942b-3ca10584d078/codes/" + v.ID,
		}

		if len(label) == 0 && len(strings.Split(e.LabelCode, ".")) != 3 {
			e.NoOfChildren = v.NoOfChildren
		}

		list = append(list, e)
	}

	return list
}

//AddChildren ...
func (r *Response) AddChildren(el []*stubs.Output) {
	r.Children = addElements(el, "")

}

//AddParent ...
func (r *Response) AddParent(el []*stubs.Output, label string) *log.Data {
	p := addElements(el, label)
	if len(p) != 1 {
		return &log.Data{"parents": p}
	}

	r.Parent = p[0]
	return nil

}

//AddLinks ...
func (r *Response) AddLinks(base string) {
	if r.Parent != nil {
		parts := strings.Split(base, "/")
		r.Parent.URL = strings.Join(parts[0:len(parts)-1], "/")
	}

	if len(strings.Split(r.LabelCode, ".")) < 2 {
		for k := range r.Children {
			r.Children[k].URL = base + "/" + r.Children[k].LabelCode
		}
	}

	r.URL = base
}
