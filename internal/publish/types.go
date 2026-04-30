package publish

import (
	"context"

	"micro-front/internal/store"
)

// Usecase は公開HTMLの生成処理を扱うユースケースです。
type Usecase struct {
	Store      *store.Store
	PublishDir string
}

// Request は静的HTML出力APIのリクエストです。
type Request struct {
	Target string `json:"target"`
	BlogID int64  `json:"blog_id"`
}

// Response は静的HTML出力APIのレスポンスです。
type Response struct {
	Result string `json:"result"`
}

// PreviewResponse はプレビュー生成APIのレスポンスです。
type PreviewResponse struct {
	Result string `json:"result"`
	URL    string `json:"url"`
}

// Run は公開HTMLの出力処理を行います。
func (uc Usecase) Run(ctx context.Context, req Request) (Response, map[string]string, error) {
	return uc.run(ctx, req)
}
