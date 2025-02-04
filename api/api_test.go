package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ONSdigital/dp-graph/v2/graph/driver"
	dbmodels "github.com/ONSdigital/dp-graph/v2/models"
	"github.com/ONSdigital/dp-hierarchy-api/datastore/datastoretest"
	"github.com/ONSdigital/dp-hierarchy-api/models"

	"github.com/gorilla/mux"
	"github.com/smartystreets/goconvey/convey"
)

const (
	hierarchyAPIURL = "http://fake-hier"
)

var router = mux.NewRouter()

func TestAPIResponseStatuses(t *testing.T) {
	t.Parallel()

	validMockDatastore := &datastoretest.StorerMock{
		GetHierarchyRootFunc: func(_ context.Context, _, _ string) (*dbmodels.HierarchyResponse, error) {
			return &dbmodels.HierarchyResponse{
				Label: "validlabel",
			}, nil
		},
		GetHierarchyElementFunc: func(_ context.Context, _, _, _ string) (*dbmodels.HierarchyResponse, error) {
			return &dbmodels.HierarchyResponse{
				Label: "validlabel",
			}, nil
		},
		GetHierarchyCodelistFunc: func(_ context.Context, _, _ string) (string, error) {
			return "codelistID", nil
		},
	}

	notFoundMockDatastore := &datastoretest.StorerMock{
		GetHierarchyRootFunc: func(_ context.Context, _, _ string) (*dbmodels.HierarchyResponse, error) {
			return nil, driver.ErrNotFound
		},
		GetHierarchyElementFunc: func(_ context.Context, _, _, _ string) (*dbmodels.HierarchyResponse, error) {
			return nil, driver.ErrNotFound
		},
		GetHierarchyCodelistFunc: func(_ context.Context, _, _ string) (string, error) {
			return "", driver.ErrNotFound
		},
	}

	convey.Convey("When asking for a hierarchy, we get a basic json response", t, func() {
		r := httptest.NewRequest("GET", "/hierarchies/hier12/dim34", http.NoBody)
		w := httptest.NewRecorder()

		api := New(router, validMockDatastore, hierarchyAPIURL)

		api.hierarchiesHandler(w, r)
		convey.So(w.Code, convey.ShouldEqual, http.StatusOK)
	})

	convey.Convey("When asking for a hierarchy node, we get a basic json response", t, func() {
		r := httptest.NewRequest("GET", "/hierarchies/hier12/dim34/codeN", http.NoBody)
		w := httptest.NewRecorder()

		api := New(router, validMockDatastore, hierarchyAPIURL)

		api.codesHandler(w, r)
		convey.So(w.Code, convey.ShouldEqual, http.StatusOK)
	})

	convey.Convey("When asking for a non-existant hierarchy, we get a 404 response", t, func() {
		r := httptest.NewRequest("GET", "/hierarchies/none/dim34", http.NoBody)
		w := httptest.NewRecorder()

		api := New(router, notFoundMockDatastore, hierarchyAPIURL)

		api.hierarchiesHandler(w, r)
		convey.So(w.Code, convey.ShouldEqual, http.StatusNotFound)
	})

	convey.Convey("When asking for a non-existant hierarchy node, we get a 404 response", t, func() {
		r := httptest.NewRequest("GET", "/hierarchies/none/dim34/codeN", http.NoBody)
		w := httptest.NewRecorder()

		api := New(router, notFoundMockDatastore, hierarchyAPIURL)

		api.codesHandler(w, r)
		convey.So(w.Code, convey.ShouldEqual, http.StatusNotFound)
	})
}

func TestMapHierarchyResponse(t *testing.T) {
	t.Parallel()

	convey.Convey("An empty DB response is mapped to an empty API response", t, func() {
		dbResponse := &dbmodels.HierarchyResponse{}
		expected := models.Response{}
		convey.So(mapHierarchyResponse(dbResponse), convey.ShouldResemble, expected)
	})

	convey.Convey("A populated DB response without children or breadcrumbs is mapped to the corresponding API response", t, func() {
		var order int64 = 123
		dbResponse := &dbmodels.HierarchyResponse{
			ID:      "testID",
			Label:   "testLabel",
			Order:   &order,
			HasData: true,
		}
		expected := models.Response{
			ID:      "testID",
			Label:   "testLabel",
			Order:   &order,
			HasData: true,
		}
		convey.So(mapHierarchyResponse(dbResponse), convey.ShouldResemble, expected)
	})

	convey.Convey("A DB response with children is mapped to the corresponding API response", t, func() {
		var order int64 = 321
		dbResponse := &dbmodels.HierarchyResponse{
			Children: []*dbmodels.HierarchyElement{
				{
					ID:      "childID",
					Label:   "childLabel",
					Order:   &order,
					HasData: true,
				},
			},
			NoOfChildren: 1,
		}
		expected := models.Response{
			Children: []*models.Element{
				{
					ID:      "childID",
					Label:   "childLabel",
					Order:   &order,
					HasData: true,
				},
			},
			NoOfChildren: 1,
		}
		convey.So(mapHierarchyResponse(dbResponse), convey.ShouldResemble, expected)
	})

	convey.Convey("A DB response with breadcrumbs is mapped to the corresponding API response", t, func() {
		var order int64 = 456
		dbResponse := &dbmodels.HierarchyResponse{
			Breadcrumbs: []*dbmodels.HierarchyElement{
				{
					ID:      "bcID",
					Label:   "bcLabel",
					Order:   &order,
					HasData: true,
				},
			},
		}
		expected := models.Response{
			Breadcrumbs: []*models.Element{
				{
					ID:      "bcID",
					Label:   "bcLabel",
					Order:   &order,
					HasData: true,
				},
			},
		}
		convey.So(mapHierarchyResponse(dbResponse), convey.ShouldResemble, expected)
	})
}
