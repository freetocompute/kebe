package responses

type RegisterSnap struct {
	Id   string `json:"snap_id"`
	Name string `json:"snap_name"`
}

type Status struct {
	Processed bool   `json:"processed"`
	Code      string `json:"code"`
	Revision  int    `json:"revision"`
}

type SnapRelease struct {
	Success bool
}
