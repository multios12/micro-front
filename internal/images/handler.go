package images

import (
	"errors"
	"net/http"
	"strconv"

	"micro-front/internal/apiutil"
	"micro-front/internal/server"
)

func (h Handler) Init(s *server.Server) {
	s.HandleFunc("GET /admin/api/blogs/{blog_id}/images", h.handlerGetImages)
	s.HandleFunc("POST /admin/api/blogs/{blog_id}/images", h.handlerPostImages)
	s.HandleFunc("DELETE /admin/api/blogs/{blog_id}/images/{image_id}", h.handlerDeleteImage)
	s.HandleFunc("GET /admin/images/{blog_id}/{image_name}", h.handlerGetImage)
}

// handlerGetImages は、記事に紐づく画像一覧を返します。
func (h Handler) handlerGetImages(w http.ResponseWriter, r *http.Request) {
	blogID, err := strconv.ParseInt(r.PathValue("blog_id"), 10, 64)
	if err != nil {
		apiutil.WriteNotFound(w, "not found")
		return
	}

	resp, err := Usecase{Store: h.Store, DataDir: h.DataDir}.List(r.Context(), blogID)
	if err != nil {
		if errors.Is(err, errImageBlogNotFound) {
			apiutil.WriteNotFound(w, "blog not found")
			return
		}
		apiutil.WriteInternalServerError(w, err)
		return
	}

	apiutil.WriteJSON(w, http.StatusOK, resp)
}

// handlerPostImages は、設計書 3.8 の画像アップロードAPIを処理します。
func (h Handler) handlerPostImages(w http.ResponseWriter, r *http.Request) {
	blogID, err := strconv.ParseInt(r.PathValue("blog_id"), 10, 64)
	if err != nil {
		apiutil.WriteNotFound(w, "not found")
		return
	}
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		apiutil.WriteValidationBodyError(w)
		return
	}

	altText := r.FormValue("alt_text")
	file, _, err := r.FormFile("file")
	if err != nil {
		apiutil.WriteValidationError(w, map[string]string{
			"file": "画像ファイルを選択してください。",
		})
		return
	}
	defer file.Close()

	resp, code, fields, err := Usecase{Store: h.Store, DataDir: h.DataDir}.Upload(r.Context(), blogID, altText, file)
	if len(fields) > 0 {
		apiutil.WriteValidationErrorCode(w, code, fields)
		return
	}
	if err != nil {
		if errors.Is(err, errImageBlogNotFound) {
			apiutil.WriteNotFound(w, "blog not found")
			return
		}
		if errors.Is(err, errImageUploadFailed) {
			apiutil.WriteError(w, http.StatusInternalServerError, "IMAGE_UPLOAD_FAILED", err.Error(), nil)
			return
		}
		apiutil.WriteInternalServerError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusCreated, resp)
}

// handlerDeleteImage は、設計書 3.9 の画像削除APIを処理します。
func (h Handler) handlerDeleteImage(w http.ResponseWriter, r *http.Request) {
	blogID, err := strconv.ParseInt(r.PathValue("blog_id"), 10, 64)
	if err != nil {
		apiutil.WriteNotFound(w, "not found")
		return
	}
	imageID, err := strconv.ParseInt(r.PathValue("image_id"), 10, 64)
	if err != nil {
		apiutil.WriteNotFound(w, "not found")
		return
	}
	resp, err := Usecase{Store: h.Store, DataDir: h.DataDir}.Delete(r.Context(), blogID, imageID)
	if err != nil {
		apiutil.WriteInternalServerError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, resp)
}

// handlerGetImage は管理画面用の画像ファイルを配信します。
func (h Handler) handlerGetImage(w http.ResponseWriter, r *http.Request) {
	blogID, err := strconv.ParseInt(r.PathValue("blog_id"), 10, 64)
	if err != nil {
		apiutil.WriteNotFound(w, "not found")
		return
	}
	path, err := Usecase{Store: h.Store, DataDir: h.DataDir}.AdminImagePath(blogID, r.PathValue("image_name"))
	if err != nil {
		apiutil.WriteNotFound(w, "not found")
		return
	}
	http.ServeFile(w, r, path)
}
