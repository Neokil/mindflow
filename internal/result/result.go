package result

// Completed acknowledges a successful operation.
type Completed struct {
	OK bool `json:"ok"`
}

// Created identifies a newly created resource.
type Created struct {
	ID string `json:"id"`
}
