package titleimage

type TemplateID string

const (
	TemplateTech    TemplateID = "tech"
	TemplateBook    TemplateID = "book"
	TemplateDiary   TemplateID = "diary"
	TemplateTravel  TemplateID = "travel"
	DefaultTemplate TemplateID = TemplateDiary
)

type GenerateInput struct {
	Title    string
	Template TemplateID
}

type Template struct {
	ID          TemplateID `json:"id"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
}
