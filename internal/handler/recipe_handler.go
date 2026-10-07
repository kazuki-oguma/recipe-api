package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"recipe-api/internal/model"
	"recipe-api/internal/repository"
)

// recipeStore はハンドラが依存する永続化層のインターフェース。
// テスト時にフェイク実装へ差し替えるために抽象化している。
type recipeStore interface {
	Create(ctx context.Context, in repository.CreateInput) (*model.Recipe, error)
	List(ctx context.Context) ([]model.Recipe, error)
	GetByID(ctx context.Context, id int64) (*model.Recipe, error)
	Update(ctx context.Context, id int64, in repository.UpdateInput) (*model.Recipe, error)
	Delete(ctx context.Context, id int64) error
}

// RecipeHandler はレシピCRUDのHTTPハンドラ群。
type RecipeHandler struct {
	store recipeStore
}

// NewRecipeHandler はRecipeHandlerを生成する。
func NewRecipeHandler(store recipeStore) *RecipeHandler {
	return &RecipeHandler{store: store}
}

// createRecipeRequest はPOST /recipesのリクエストボディ。
// 未指定とゼロ値を区別するためポインタ型にしている。
type createRecipeRequest struct {
	Title       *string `json:"title"`
	MakingTime  *string `json:"making_time"`
	Serves      *string `json:"serves"`
	Ingredients *string `json:"ingredients"`
	Cost        *int    `json:"cost"`
}

// requiredFieldsMessage はPOST /recipesのバリデーションエラー時に返す必須項目一覧。
const requiredFieldsMessage = "title, making_time, serves, ingredients, cost"

// hasMissingFields は必須項目のいずれかが未指定・空文字かどうかを返す。
func (req createRecipeRequest) hasMissingFields() bool {
	return req.Title == nil || strings.TrimSpace(*req.Title) == "" ||
		req.MakingTime == nil || strings.TrimSpace(*req.MakingTime) == "" ||
		req.Serves == nil || strings.TrimSpace(*req.Serves) == "" ||
		req.Ingredients == nil || strings.TrimSpace(*req.Ingredients) == "" ||
		req.Cost == nil
}

// CreateRecipe は POST /recipes を処理する。
func (h *RecipeHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req createRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"message":  "Recipe creation failed!",
			"required": requiredFieldsMessage,
		})
		return
	}

	if req.hasMissingFields() {
		writeJSON(w, http.StatusOK, map[string]any{
			"message":  "Recipe creation failed!",
			"required": requiredFieldsMessage,
		})
		return
	}

	recipe, err := h.store.Create(r.Context(), repository.CreateInput{
		Title:       *req.Title,
		MakingTime:  *req.MakingTime,
		Serves:      *req.Serves,
		Ingredients: *req.Ingredients,
		Cost:        *req.Cost,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"message": "レシピの作成に失敗しました",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Recipe successfully created!",
		"recipe":  []model.Recipe{*recipe},
	})
}

// ListRecipes は GET /recipes を処理する。データベース内の全レシピを返す。
func (h *RecipeHandler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	recipes, err := h.store.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"message": "レシピ一覧の取得に失敗しました",
		})
		return
	}

	summaries := make([]model.RecipeSummary, 0, len(recipes))
	for _, recipe := range recipes {
		summaries = append(summaries, recipe.ToSummary())
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"recipes": summaries,
	})
}

// GetRecipe は GET /recipes/{id} を処理する。
func (h *RecipeHandler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	recipe, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"message": "recipe not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"message": "レシピの取得に失敗しました",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Recipe details by id",
		"recipe":  []model.RecipeSummary{recipe.ToSummary()},
	})
}

// updateRecipeRequest はPATCH /recipes/{id}のリクエストボディ。全項目任意（部分更新）。
type updateRecipeRequest struct {
	Title       *string `json:"title"`
	MakingTime  *string `json:"making_time"`
	Serves      *string `json:"serves"`
	Ingredients *string `json:"ingredients"`
	Cost        *int    `json:"cost"`
}

// UpdateRecipe は PATCH /recipes/{id} を処理する。
func (h *RecipeHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req updateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "リクエストボディの形式が不正です",
		})
		return
	}

	if req.Title == nil && req.MakingTime == nil && req.Serves == nil && req.Ingredients == nil && req.Cost == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "更新する項目が指定されていません",
		})
		return
	}

	recipe, err := h.store.Update(r.Context(), id, repository.UpdateInput{
		Title:       req.Title,
		MakingTime:  req.MakingTime,
		Serves:      req.Serves,
		Ingredients: req.Ingredients,
		Cost:        req.Cost,
	})
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"message": "recipe not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"message": "レシピの更新に失敗しました",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Recipe successfully updated!",
		"recipe":  []model.RecipePatchResult{recipe.ToPatchResult()},
	})
}

// DeleteRecipe は DELETE /recipes/{id} を処理する。
func (h *RecipeHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	err := h.store.Delete(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"message": "No Recipe found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"message": "レシピの削除に失敗しました",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Recipe successfully removed!",
	})
}

// parseID はパスパラメータ{id}を数値へ変換する。
// {id}自体は公開済みエンドポイントの一部であるため、形式が不正な場合も200で返す
// （404は「リソース未存在」「未公開エンドポイント」のみに用いる）。
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "idは数値で指定してください",
		})
		return 0, false
	}
	return id, true
}
