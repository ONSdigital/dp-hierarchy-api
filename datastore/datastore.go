package datastore

import (
	"context"
	dbmodels "github.com/ONSdigital/dp-graph/v2/models"
)

//go:generate moq -out datastoretest/storer.go -pkg datastoretest . Storer

// Storer is the generic interface for the database
type Storer interface {
	Close(ctx context.Context) error
	GetHierarchyCodelist(ctx context.Context, instanceID, dimension string) (string, error)
	GetHierarchyRoot(ctx context.Context, instanceID, dimension string) (*dbmodels.HierarchyResponse, error)
	GetHierarchyElement(ctx context.Context, instanceID, dimension, code string) (*dbmodels.HierarchyResponse, error)
}
