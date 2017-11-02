package models

import "context"

// Storer is the generic interface for the database
type Storer interface {
	Close(ctx context.Context) error
	GetCodelist(hierarchy *Hierarchy) (string, error)
	GetHierarchy(hierarchy *Hierarchy) (*Response, error)
	GetCode(hierarchy *Hierarchy, code string) (*Response, error)
	Ping(ctx context.Context) error
}
