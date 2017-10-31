package models

// Hierarchy models a specific hierarchy (instance, dimension, url and codelist)
type Hierarchy struct {
	InstanceId string
	Dimension  string
	URL        string
	CodelistId string
}
