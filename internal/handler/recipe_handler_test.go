package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"recipe-api/internal/model"
	"recipe-api/internal/repository"
)

// fakeStore はrecipeStoreインターフェースを満たすテスト用のインメモリ実装。
// Go標準のtestingパッケージを用いるため、JUnitに相当する正常系・異常系・境界値を
// テーブル駆動ではなく個別テスト関数として記述する。
type fakeStore struct {
	createFn func(ctx context.Context, in repository.CreateInput) (*model.Recipe, error)
	listFn   func(ctx context.Context) ([]model.Recipe, error)
	getFn    func(ctx context.Context, id int64) (*model.Recipe, error)
	updateFn func(ctx context.Context, id int64, in repository.UpdateInput) (*model.Recipe, error)
	deleteFn func(ctx context.Context, id int64) error
}

func (f *fakeStore) Create(ctx context.Context, in repository.CreateInput) (*model.Recipe, error) {
	return f.createFn(ctx, in)
}

func (f *fakeStore) List(ctx context.Context) ([]model.Recipe, error) {
	return f.listFn(ctx)
}

func (f *fakeStore) GetByID(ctx context.Context, id int64) (*model.Recipe, error) {
	return f.getFn(ctx, id)
}

func (f *fakeStore) Update(ctx context.Context, id int64, in repository.UpdateInput) (*model.Recipe, error) {
	return f.updateFn(ctx, id, in)
}

func (f *fakeStore) Delete(ctx context.Context, id int64) error {
	return f.deleteFn(ctx, id)
}

func newTestMux(store recipeStore) *http.ServeMux {
	h := NewRecipeHandler(store)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /recipes", h.CreateRecipe)
	mux.HandleFunc("GET /recipes", h.ListRecipes)
	mux.HandleFunc("GET /recipes/{id}", h.GetRecipe)
	mux.HandleFunc("PATCH /recipes/{id}", h.UpdateRecipe)
	mux.HandleFunc("DELETE /recipes/{id}", h.DeleteRecipe)
	return mux
}

// 正常系: 必須項目が揃っていれば200で、recipeは1件の配列・costは文字列として返される。
func TestCreateRecipe_Success(t *testing.T) {
	store := &fakeStore{
		createFn: func(ctx context.Context, in repository.CreateInput) (*model.Recipe, error) {
			return &model.Recipe{
				ID:          3,
				Title:       in.Title,
				MakingTime:  in.MakingTime,
				Serves:      in.Serves,
				Ingredients: in.Ingredients,
				Cost:        model.Cost(in.Cost),
				CreatedAt:   "2026-10-07 00:00:00",
				UpdatedAt:   "2026-10-07 00:00:00",
			}, nil
		},
	}
	mux := newTestMux(store)

	body := bytes.NewBufferString(`{"title":"カレー","making_time":"30分","serves":"2人","ingredients":"玉ねぎ","cost":500}`)
	req := httptest.NewRequest(http.MethodPost, "/recipes", body)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got["message"] != "Recipe successfully created!" {
		t.Errorf("message = %v", got["message"])
	}
	recipes, ok := got["recipe"].([]any)
	if !ok || len(recipes) != 1 {
		t.Fatalf("recipe = %v, want array with 1 element", got["recipe"])
	}
	recipe, ok := recipes[0].(map[string]any)
	if !ok {
		t.Fatalf("recipe[0] = %v, want object", recipes[0])
	}
	if cost, ok := recipe["cost"].(string); !ok || cost != "500" {
		t.Errorf("cost = %v, want string \"500\"", recipe["cost"])
	}
}

// 異常系: 必須項目が不足している場合も200を返し、固定のrequiredメッセージを含める
// (test/test.rest.js の公式テストスイートが200を期待するため)。
func TestCreateRecipe_MissingFields(t *testing.T) {
	mux := newTestMux(&fakeStore{})

	body := bytes.NewBufferString(`{"title":"カレー"}`)
	req := httptest.NewRequest(http.MethodPost, "/recipes", body)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got["message"] != "Recipe creation failed!" {
		t.Errorf("message = %v", got["message"])
	}
	if got["required"] != "title, making_time, serves, ingredients, cost" {
		t.Errorf("required = %v", got["required"])
	}
}

// 境界値: costが0でも必須項目を満たしていれば作成できる(0は「未指定」とは区別される)。
func TestCreateRecipe_CostZeroBoundary(t *testing.T) {
	store := &fakeStore{
		createFn: func(ctx context.Context, in repository.CreateInput) (*model.Recipe, error) {
			if in.Cost != 0 {
				t.Fatalf("cost = %d, want 0", in.Cost)
			}
			return &model.Recipe{ID: 1, Title: in.Title, MakingTime: in.MakingTime, Serves: in.Serves, Ingredients: in.Ingredients, Cost: model.Cost(in.Cost)}, nil
		},
	}
	mux := newTestMux(store)

	body := bytes.NewBufferString(`{"title":"水","making_time":"1分","serves":"1人","ingredients":"水","cost":0}`)
	req := httptest.NewRequest(http.MethodPost, "/recipes", body)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// 正常系: データベース内の全レシピを一覧取得し、costは文字列・タイムスタンプは含まれない。
func TestListRecipes_Success(t *testing.T) {
	store := &fakeStore{
		listFn: func(ctx context.Context) ([]model.Recipe, error) {
			return []model.Recipe{
				{ID: 1, Title: "チキンカレー", MakingTime: "45分", Serves: "4人", Ingredients: "玉ねぎ,肉,スパイス", Cost: 1000, CreatedAt: "2016-01-10 12:10:12", UpdatedAt: "2016-01-10 12:10:12"},
				{ID: 2, Title: "オムライス", MakingTime: "30分", Serves: "2人", Ingredients: "玉ねぎ,卵,スパイス,醤油", Cost: 700, CreatedAt: "2016-01-11 13:10:12", UpdatedAt: "2016-01-11 13:10:12"},
			}, nil
		},
	}
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/recipes", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	recipes, ok := got["recipes"].([]any)
	if !ok || len(recipes) != 2 {
		t.Fatalf("recipes = %v, want array with 2 elements", got["recipes"])
	}
	first, ok := recipes[0].(map[string]any)
	if !ok {
		t.Fatalf("recipes[0] = %v, want object", recipes[0])
	}
	if cost, ok := first["cost"].(string); !ok || cost != "1000" {
		t.Errorf("cost = %v, want string \"1000\"", first["cost"])
	}
	if _, exists := first["created_at"]; exists {
		t.Errorf("created_at should not be present in list response, got %v", first["created_at"])
	}
	if _, exists := first["updated_at"]; exists {
		t.Errorf("updated_at should not be present in list response, got %v", first["updated_at"])
	}
}

// 異常系: データベースアクセスに失敗した場合は500を返す。
func TestListRecipes_Error(t *testing.T) {
	store := &fakeStore{
		listFn: func(ctx context.Context) ([]model.Recipe, error) {
			return nil, errors.New("db error")
		},
	}
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/recipes", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// 正常系: 存在するIDを取得すると200でmessageと1件のrecipe配列を返す(タイムスタンプは含まない)。
func TestGetRecipe_Found(t *testing.T) {
	store := &fakeStore{
		getFn: func(ctx context.Context, id int64) (*model.Recipe, error) {
			return &model.Recipe{
				ID: id, Title: "オムライス", MakingTime: "30分", Serves: "2人",
				Ingredients: "玉ねぎ,卵,スパイス,醤油", Cost: 700,
				CreatedAt: "2016-01-11 13:10:12", UpdatedAt: "2016-01-11 13:10:12",
			}, nil
		},
	}
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/recipes/2", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got["message"] != "Recipe details by id" {
		t.Errorf("message = %v", got["message"])
	}
	recipes, ok := got["recipe"].([]any)
	if !ok || len(recipes) != 1 {
		t.Fatalf("recipe = %v, want array with 1 element", got["recipe"])
	}
	recipe, ok := recipes[0].(map[string]any)
	if !ok {
		t.Fatalf("recipe[0] = %v, want object", recipes[0])
	}
	if cost, ok := recipe["cost"].(string); !ok || cost != "700" {
		t.Errorf("cost = %v, want string \"700\"", recipe["cost"])
	}
	if _, exists := recipe["created_at"]; exists {
		t.Errorf("created_at should not be present, got %v", recipe["created_at"])
	}
}

// 異常系: 存在しないIDを取得すると404を返す。
func TestGetRecipe_NotFound(t *testing.T) {
	store := &fakeStore{
		getFn: func(ctx context.Context, id int64) (*model.Recipe, error) {
			return nil, repository.ErrNotFound
		},
	}
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/recipes/999", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// 境界値: idが数値でない場合も公開済みエンドポイントのため200を返す。
func TestGetRecipe_InvalidID(t *testing.T) {
	mux := newTestMux(&fakeStore{})

	req := httptest.NewRequest(http.MethodGet, "/recipes/abc", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// 異常系: 未公開のエンドポイントへアクセスすると404を返す。
func TestUndefinedEndpoint_NotFound(t *testing.T) {
	mux := newTestMux(&fakeStore{})

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// 正常系: DELETEが成功すると200を返す。
func TestDeleteRecipe_Success(t *testing.T) {
	store := &fakeStore{
		deleteFn: func(ctx context.Context, id int64) error { return nil },
	}
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodDelete, "/recipes/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// 異常系: 存在しないIDをDELETEすると404と"No Recipe found"を返す。
func TestDeleteRecipe_NotFound(t *testing.T) {
	store := &fakeStore{
		deleteFn: func(ctx context.Context, id int64) error { return repository.ErrNotFound },
	}
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodDelete, "/recipes/999", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got["message"] != "No Recipe found" {
		t.Errorf("message = %v", got["message"])
	}
}

// 正常系: PATCHで一部項目のみ更新でき、recipeはidを含まない1件の配列で返る。
func TestUpdateRecipe_PartialSuccess(t *testing.T) {
	store := &fakeStore{
		updateFn: func(ctx context.Context, id int64, in repository.UpdateInput) (*model.Recipe, error) {
			if in.Title == nil || *in.Title != "トマトスープレシピ" {
				t.Fatalf("title = %v, want トマトスープレシピ", in.Title)
			}
			if in.Cost != nil {
				t.Fatalf("cost should remain unspecified, got %v", *in.Cost)
			}
			return &model.Recipe{
				ID: id, Title: *in.Title, MakingTime: "15分", Serves: "5人",
				Ingredients: "玉ねぎ, トマト, スパイス, 水", Cost: 450,
			}, nil
		},
	}
	mux := newTestMux(store)

	body := bytes.NewBufferString(`{"title":"トマトスープレシピ"}`)
	req := httptest.NewRequest(http.MethodPatch, "/recipes/1", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got["message"] != "Recipe successfully updated!" {
		t.Errorf("message = %v", got["message"])
	}
	recipes, ok := got["recipe"].([]any)
	if !ok || len(recipes) != 1 {
		t.Fatalf("recipe = %v, want array with 1 element", got["recipe"])
	}
	recipe, ok := recipes[0].(map[string]any)
	if !ok {
		t.Fatalf("recipe[0] = %v, want object", recipes[0])
	}
	if _, exists := recipe["id"]; exists {
		t.Errorf("id should not be present in patch response, got %v", recipe["id"])
	}
	if cost, ok := recipe["cost"].(string); !ok || cost != "450" {
		t.Errorf("cost = %v, want string \"450\"", recipe["cost"])
	}
}

// 異常系: PATCHで更新項目が何も指定されていない場合も200を返す。
func TestUpdateRecipe_NoFields(t *testing.T) {
	mux := newTestMux(&fakeStore{})

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPatch, "/recipes/1", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
