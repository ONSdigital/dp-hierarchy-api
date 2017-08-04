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
	Parent    *Elements   `json:"parents,omitempty"`
	//CodeURL      string `json:"code_url"`
}

//Elements ...
type Elements struct {
	ID           string `json:"id"`
	LabelCode    string `json:"label_code"`
	Label        string `json:"label"`
	NoOfChildren int    `json:"number_of_children,omitempty"`
	//CodeURL      string `json:"code_url"`
}

func addElements(el []*stubs.Output, label string) []*Elements {
	var list []*Elements
	for _, v := range el {

		//remove wrongful parent elements
		if len(label) != 0 && (len(strings.Split(label, ".")) < len(strings.Split(v.LabelCode, "."))) {
			continue
		}

		e := &Elements{
			ID:        v.ID,
			Label:     v.Label,
			LabelCode: v.LabelCode,
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
