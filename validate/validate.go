package validate

import (
	"net/http"
	"strings"

	"github.com/ONSdigital/dp-hierarchy-api/stubs"
	"github.com/ONSdigital/go-ns/log"
)

//Request ...
type Request struct {
	R  *http.Request
	W  http.ResponseWriter
	ID string
}

var hierarchy = stubs.GenerateHierarchy()

//Validate ...
func (r *Request) Validate(levels ...string) bool {
	if len(levels) == 0 {
		//not enough args error
		return false
	}

	if ok := r.validateLevel1(levels[0]); !ok {
		return false
	}

	if len(levels) > 1 {
		if ok := r.validateLevel2(levels[0], levels[1]); !ok {
			return false
		}
	}

	return true
}

func (r *Request) validateLevel1(level string) bool {
	for _, c := range hierarchy[r.ID].Children {
		if c.LabelCode == level {
			return true
		}
	}
	return false
}

func (r *Request) validateLevel2(level1, level2 string) bool {
	label := level1 + "." + level2
	for _, c := range hierarchy[level1].Children {
		l := strings.Split(c.LabelCode, ".")
		if len(l) != 2 {
			log.DebugR(r.R, "invalid child element found", log.Data{"id": r.ID, "level2 label": label, "childLabel": c.LabelCode})
			r.W.WriteHeader(http.StatusBadRequest)
			return false
		}

		if l[1] == level2 {
			return true
		}
	}

	log.DebugR(r.R, "second level hierarchy element not found", log.Data{"id": r.ID, "level2 label": label})
	r.W.WriteHeader(http.StatusNotFound)
	return false
}
