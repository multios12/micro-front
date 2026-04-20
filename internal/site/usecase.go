package site

import (
	"context"
	"strconv"

	"micro-front/internal/store"
	"micro-front/internal/validate"
)

// Get は設計書 3.1 のサイト情報取得処理を行います。
func (uc Usecase) Get(ctx context.Context) (SiteGetResponse, error) {
	settings, err := uc.Store.GetSiteSettings(ctx)
	if err != nil {
		return SiteGetResponse{}, err
	}
	return SiteGetResponse(settings), nil
}

// Put はサイト情報を検証して保存し、更新後の内容を返します。
func (uc Usecase) Put(ctx context.Context, req SitePutRequest) (SitePutResponse, string, map[string]string, error) {
	code, fields := validateSiteSettings(req)
	if len(fields) > 0 {
		return SitePutResponse{}, code, fields, nil
	}

	updated, err := uc.Store.UpdateSiteSettings(ctx, store.SiteEntitty{
		SiteTitle:       req.SiteTitle,
		SiteSubtitle:    req.SiteSubtitle,
		SiteDescription: req.SiteDescription,
		Tabs:            req.Tabs,
		FootInformation: req.FootInformation,
		Copyright:       req.Copyright,
	})
	if err != nil {
		return SitePutResponse{}, "", nil, err
	}

	return SitePutResponse(updated), "", nil, nil
}

// validateSiteSettings はサイト情報更新APIの入力値を検証します。
func validateSiteSettings(req SitePutRequest) (string, map[string]string) {
	fields := map[string]string{}
	code := "VALIDATION_ERROR"
	if validate.Length(req.SiteTitle) == 0 {
		fields["site_title"] = "タイトルを入力してください。"
	} else if validate.Length(req.SiteTitle) > 100 {
		fields["site_title"] = "タイトルは100文字以内で入力してください。"
	}

	if validate.Length(req.SiteSubtitle) == 0 {
		fields["site_subtitle"] = "サブタイトルを入力してください。"
	} else if validate.Length(req.SiteSubtitle) > 100 {
		fields["site_subtitle"] = "サブタイトルは100文字以内で入力してください。"
	}

	if validate.Length(req.SiteDescription) == 0 {
		fields["site_description"] = "サイト説明を入力してください。"
	} else if validate.Length(req.SiteDescription) > 1000 {
		fields["site_description"] = "サイト説明は1000文字以内で入力してください。"
	}

	if len(req.Tabs) == 0 {
		fields["tabs"] = "タブを追加してください。"
	}
	if len(req.Tabs) > 10 {
		fields["tabs"] = "タブは10件以内で入力してください。"
	}
	for i, tab := range req.Tabs {
		if validate.Length(tab.TabLabel) == 0 {
			fields["tabs["+strconv.Itoa(i)+"].tab_label"] = "タブラベルを入力してください。"
		} else if validate.Length(tab.TabLabel) > 20 {
			fields["tabs["+strconv.Itoa(i)+"].tab_label"] = "タブラベルは20文字以内で入力してください。"
		}
		if validate.Length(tab.TabURL) == 0 {
			fields["tabs["+strconv.Itoa(i)+"].tab_url"] = "タブURLを入力してください。"
		} else if validate.Length(tab.TabURL) > 100 {
			fields["tabs["+strconv.Itoa(i)+"].tab_url"] = "タブURLは100文字以内で入力してください。"
		} else if len(tab.TabURL) == 0 || tab.TabURL[0] != '/' {
			fields["tabs["+strconv.Itoa(i)+"].tab_url"] = "タブURLの形式が不正です。"
			code = "INVALID_TAB_URL"
		}
	}

	if validate.Length(req.FootInformation) == 0 {
		fields["foot_information"] = "フッタ情報を入力してください。"
	} else if validate.Length(req.FootInformation) > 100 {
		fields["foot_information"] = "フッタ情報は100文字以内で入力してください。"
	}

	if validate.Length(req.Copyright) == 0 {
		fields["copyright"] = "コピーライトを入力してください。"
	} else if validate.Length(req.Copyright) > 100 {
		fields["copyright"] = "コピーライトは100文字以内で入力してください。"
	}
	return code, fields
}
