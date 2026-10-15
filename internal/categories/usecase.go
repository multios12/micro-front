package categories

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"micro-front/internal/store"
)

func (uc Usecase) List(ctx context.Context) (ListResponse, error) {
	items, err := uc.Store.ListCategories(ctx)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Items: items}, nil
}

func (uc Usecase) Create(ctx context.Context, req CreateRequest) (store.CategoryEntity, map[string]string, error) {
	fields := map[string]string{}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		fields["name"] = "カテゴリ名を入力してください。"
	}
	if req.SortOrder < 0 {
		fields["sort_order"] = "表示順は0以上の整数で入力してください。"
	}
	isHeading := req.ParentID == nil
	if req.IsHeading != nil {
		isHeading = *req.IsHeading
	}
	if isHeading && req.ParentID != nil {
		fields["is_heading"] = "親見出しには親見出しを指定できません。"
	}
	if req.ParentID != nil {
		parent, err := uc.Store.GetCategory(ctx, *req.ParentID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && parent.ParentID != nil) {
			fields["parent_id"] = "親には登録済みの親見出しを指定してください。"
		} else if err != nil {
			return store.CategoryEntity{}, nil, err
		}
	}
	if len(fields) > 0 {
		return store.CategoryEntity{}, fields, nil
	}
	item, err := uc.Store.CreateCategory(ctx, store.CategoryEntity{
		ParentID: req.ParentID, Name: req.Name, SortOrder: req.SortOrder,
		IsHeading: isHeading,
	})
	return item, nil, err
}

func (uc Usecase) Delete(ctx context.Context, id int64) error {
	return uc.Store.DeleteCategory(ctx, id)
}

func (uc Usecase) Move(ctx context.Context, id int64, direction string) error {
	return uc.Store.MoveCategory(ctx, id, direction)
}

// Update は名前と表示順を更新し、親子関係を維持します。
func (uc Usecase) Update(ctx context.Context, id int64, req UpdateRequest) (store.CategoryEntity, map[string]string, error) {
	item, err := uc.Store.GetCategory(ctx, id)
	if err != nil {
		return store.CategoryEntity{}, nil, err
	}
	fields := map[string]string{}
	item.Name = strings.TrimSpace(req.Name)
	item.SortOrder = req.SortOrder
	if item.Name == "" {
		fields["name"] = "カテゴリ名を入力してください。"
	}
	if item.SortOrder < 0 {
		fields["sort_order"] = "表示順は0以上の整数で入力してください。"
	}
	if len(fields) > 0 {
		return store.CategoryEntity{}, fields, nil
	}
	item, err = uc.Store.UpdateCategory(ctx, id, item)
	return item, nil, err
}
