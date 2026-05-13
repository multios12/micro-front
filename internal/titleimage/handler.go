package titleimage

import (
	"net/http"

	"micro-front/internal/apiutil"
	"micro-front/internal/server"
)

type Handler struct{}

type templatesResponse struct {
	Items []Template `json:"items"`
}

type previewRequest struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Template string `json:"template"`
}

type previewResponse struct {
	SVG string `json:"svg"`
}

func (h Handler) Init(s *server.Server) {
	s.HandleFunc("GET /admin/api/title-image/templates", h.handleGetTemplates)
	s.HandleFunc("POST /admin/api/title-image/preview", h.handlePostPreview)
}

func (h Handler) handleGetTemplates(w http.ResponseWriter, r *http.Request) {
	apiutil.WriteJSON(w, http.StatusOK, templatesResponse{Items: ListTemplates()})
}

func (h Handler) handlePostPreview(w http.ResponseWriter, r *http.Request) {
	var req previewRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteValidationBodyError(w)
		return
	}
	svg, err := GenerateSVG(GenerateInput{
		Title:    req.Title,
		Category: req.Category,
		Template: TemplateID(req.Template),
	})
	if err != nil {
		apiutil.WriteValidationErrorCode(w, "INVALID_TITLE_IMAGE_TEMPLATE", map[string]string{
			"template": "タイトル画像テンプレートが不正です。",
		})
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, previewResponse{SVG: svg})
}
