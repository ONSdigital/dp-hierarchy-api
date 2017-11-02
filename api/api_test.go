package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

type fakedb struct {
}

const (
	hierarchyAPIURL = "http://fake-hier"
	codelistURL     = "http://fake-codelist"
)

var router = mux.NewRouter()

func (db fakedb) Close(ctx context.Context) error {
	return nil
}
func (db fakedb) GetCode(h *models.Hierarchy, code string) (*models.Response, error) {
	res := &models.Response{
		Label: "lay-bull",
		ID:    code,
		Breadcrumbs: []*models.Element{
			&models.Element{
				Label:        "child1",
				NoOfChildren: 1,
			},
		},
	}
	return res, nil
}
func (db fakedb) GetCodelist(h *models.Hierarchy) (string, error) {
	if h.InstanceId == "fail" {
		return "", errors.New("we failed you, sorry")
	} else if h.InstanceId == "none" {
		return "", nil
	}
	return "clistABC", nil
}
func (db fakedb) GetHierarchy(h *models.Hierarchy) (*models.Response, error) {
	res := &models.Response{
		Label: "h-lay-bull",
		ID:    "h-eye-dee",
		Children: []*models.Element{
			&models.Element{
				Label:        "h-child1",
				NoOfChildren: 2,
			},
		},
	}
	return res, nil
}
func (db fakedb) Ping(ctx context.Context) error {
	return nil
}

func TestSanity(t *testing.T) {
	t.Parallel()

	SetDatabase(fakedb{})
	HierarchyAPIURL = hierarchyAPIURL
	models.CodelistURL = codelistURL
	AddRoutes(router)

	Convey("When asking for a hierarchy, we get a basic json response", t, func() {
		r := httptest.NewRequest("GET", "/hierarchies/hier12/dim34", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.String(), ShouldEqual,
			`{"label":"h-lay-bull",`+
				`"children":[{"label":"h-child1","no_of_children":2,"links":{"codelist":{"href":"`+codelistURL+`/code-list/clistABC/code"},"self":{"href":"`+hierarchyAPIURL+`/hierarchies/hier12/dim34"}},"has_data":false}],`+
				`"links":{"codelist":{"id":"h-eye-dee","href":"`+codelistURL+`/code-list/clistABC/code/h-eye-dee"},"self":{"id":"h-eye-dee","href":"`+hierarchyAPIURL+`/hierarchies/hier12/dim34"}}`+
				`}`,
		)
	})
	Convey("When asking for a hierarchy node, we get a basic json response", t, func() {
		r := httptest.NewRequest("GET", "/hierarchies/hier12/dim34/codeN", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.String(), ShouldEqual,
			`{"label":"lay-bull",`+
				`"links":{"codelist":{"id":"codeN","href":"`+codelistURL+`/code-list/clistABC/code/codeN"},"self":{"id":"codeN","href":"`+hierarchyAPIURL+`/hierarchies/hier12/dim34/codeN"}},`+
				`"breadcrumbs":[{"label":"child1","no_of_children":1,"has_data":false}]`+
				`}`,
		)
	})
	Convey("When asking for a failure hierarchy code, we get a server error", t, func() {
		r := httptest.NewRequest("GET", "/hierarchies/fail/dim34/codeN", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})
	Convey("When asking for a non-existant hierarchy, we get a 404 response", t, func() {
		r := httptest.NewRequest("GET", "/hierarchies/none/dim34/codeN", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
	})
}
