package templates

type File struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
}

type Template struct {
	Templates []File `json:"templates"`
	Values    Values `json:"values"`
}
