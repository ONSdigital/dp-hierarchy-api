package stubs

import (
	"strings"
)

const cpi = "CPI"

//Output contains details for a specific element in a hierarchy, including
//relationships to other elements
type Output struct {
	ID           string    `json:"id"`
	Level        int       `json:"level"`
	LabelCode    string    `json:"label_code"`
	Label        string    `json:"label"`
	Children     []*Output `json:"children,omitempty"`
	Parents      []*Output `json:"parents,omitempty"`
	NoOfChildren int       `json:"number_of_children,omitempty"`
}

func (o *Output) addChildren(child *Output) {
	child.Parents = nil
	child.Children = nil

	o.Children = append(o.Children, child)
}
func (o *Output) addParents(parent *Output) {
	parent.Parents = nil
	parent.Children = nil

	o.Parents = append(o.Parents, parent)
}

//GenerateHierarchy uses a json file to produce a single hierarchy structure in a map
func GenerateHierarchy() map[string]*Output {
	codes := GoodsAndServicesIndices.Codes

	//for each codeEntry
	webLabels := make(map[string]*Output)
	for _, code := range codes {
		l := strings.Split(code.Label, " ")
		webLabels[l[0]] = &Output{
			ID:        code.ID,
			LabelCode: l[0],
			Label:     strings.Join(l[1:], " "),
		}
	}

	webLabels[cpi].Level = 0

	for k, v := range webLabels {
		if strings.Contains(v.LabelCode, ".") || k == cpi {
			continue
		}

		webLabels[k].Level = 1
		webLabels[cpi].Children = append(webLabels[cpi].Children, v)
		webLabels[k].Parents = append(webLabels[k].Parents, webLabels[cpi])
	}

	webLabels[cpi].NoOfChildren = len(webLabels[cpi].Children)

	for bi, base := range webLabels {
		if base.LabelCode == cpi {
			continue
		}

		for ci, compare := range webLabels {
			if (!strings.Contains(ci, bi) && !strings.Contains(bi, ci)) || ci == bi {
				continue
			}

			compareLevel := len(strings.Split(ci, "."))
			webLabels[ci].Level = compareLevel

			if (compareLevel - base.Level) == 1 {
				webLabels[bi].NoOfChildren++
				webLabels[bi].Children = upsert(webLabels[bi].Children, compare)
				webLabels[ci].Parents = upsert(webLabels[ci].Parents, base)
			}
		}

	}

	return webLabels
}

func upsert(c []*Output, item *Output) []*Output {
	for _, v := range c {
		if v == item {
			return c
		}
	}

	return append(c, item)
}
