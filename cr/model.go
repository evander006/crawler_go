package cr

type Page struct {
	Resource string  `json:"resource"`
	Title    string  `json:"title"`
	Links    []*Page `json:"links"`
}

func NewPage(resource, title string) *Page {
	return &Page{
		Resource: resource,
		Title:    title,
		Links:    []*Page{},
	}
}
