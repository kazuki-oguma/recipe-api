// Package repository はrecipesテーブルへの永続化処理を提供する。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"recipe-api/internal/model"
)

// ErrNotFound は指定したIDのレシピが存在しない場合に返すエラー。
var ErrNotFound = errors.New("recipe not found")

// timeLayout はTIMESTAMP列を文字列化する際のフォーマット。
const timeLayout = "2006-01-02 15:04:05"

// RecipeRepository はrecipesテーブルへのCRUD操作を提供する。
type RecipeRepository struct {
	db *sql.DB
}

// NewRecipeRepository はRecipeRepositoryを生成する。
func NewRecipeRepository(db *sql.DB) *RecipeRepository {
	return &RecipeRepository{db: db}
}

// CreateInput は作成時に必須の入力値。
type CreateInput struct {
	Title       string
	MakingTime  string
	Serves      string
	Ingredients string
	Cost        int
}

// Create は新しいレシピを1件登録し、登録後のレコードを返す。
// PostgreSQLはLastInsertId()をサポートしないため、RETURNING句で登録結果を直接取得する。
func (r *RecipeRepository) Create(ctx context.Context, in CreateInput) (*model.Recipe, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO recipes (title, making_time, serves, ingredients, cost)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, title, making_time, serves, ingredients, cost, created_at, updated_at`,
		in.Title, in.MakingTime, in.Serves, in.Ingredients, in.Cost,
	)
	recipe, err := scanRecipe(row)
	if err != nil {
		return nil, fmt.Errorf("insert recipe: %w", err)
	}
	return &recipe, nil
}

// List は全レシピをID昇順で返す。
func (r *RecipeRepository) List(ctx context.Context) ([]model.Recipe, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, making_time, serves, ingredients, cost, created_at, updated_at FROM recipes ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("select recipes: %w", err)
	}
	defer rows.Close()

	recipes := make([]model.Recipe, 0)
	for rows.Next() {
		recipe, err := scanRecipe(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recipe: %w", err)
		}
		recipes = append(recipes, recipe)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipes: %w", err)
	}
	return recipes, nil
}

// GetByID はIDを指定して1件取得する。存在しない場合はErrNotFoundを返す。
func (r *RecipeRepository) GetByID(ctx context.Context, id int64) (*model.Recipe, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, title, making_time, serves, ingredients, cost, created_at, updated_at FROM recipes WHERE id = $1`,
		id,
	)
	recipe, err := scanRecipe(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan recipe: %w", err)
	}
	return &recipe, nil
}

// UpdateInput はPATCH時に指定されたフィールドのみを更新するための入力。
// nilのフィールドは更新対象外（部分更新）。
type UpdateInput struct {
	Title       *string
	MakingTime  *string
	Serves      *string
	Ingredients *string
	Cost        *int
}

// Update は指定されたフィールドのみを更新し、RETURNING句で更新後のレコードを直接取得する。
// 対象IDが存在しない場合はErrNotFoundを返す。
func (r *RecipeRepository) Update(ctx context.Context, id int64, in UpdateInput) (*model.Recipe, error) {
	sets := make([]string, 0, 5)
	args := make([]any, 0, 6)
	argIdx := 1

	addSet := func(column string, value any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", column, argIdx))
		args = append(args, value)
		argIdx++
	}

	if in.Title != nil {
		addSet("title", *in.Title)
	}
	if in.MakingTime != nil {
		addSet("making_time", *in.MakingTime)
	}
	if in.Serves != nil {
		addSet("serves", *in.Serves)
	}
	if in.Ingredients != nil {
		addSet("ingredients", *in.Ingredients)
	}
	if in.Cost != nil {
		addSet("cost", *in.Cost)
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE recipes SET %s WHERE id = $%d
		 RETURNING id, title, making_time, serves, ingredients, cost, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx,
	)

	row := r.db.QueryRowContext(ctx, query, args...)
	recipe, err := scanRecipe(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update recipe: %w", err)
	}
	return &recipe, nil
}

// Delete はIDを指定してレシピを削除する。対象が存在しない場合はErrNotFoundを返す。
func (r *RecipeRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM recipes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete recipe: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// rowScanner は *sql.Row と *sql.Rows の共通インターフェース。
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecipe(s rowScanner) (model.Recipe, error) {
	var (
		recipe    model.Recipe
		createdAt time.Time
		updatedAt time.Time
	)
	err := s.Scan(
		&recipe.ID,
		&recipe.Title,
		&recipe.MakingTime,
		&recipe.Serves,
		&recipe.Ingredients,
		&recipe.Cost,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return model.Recipe{}, err
	}
	recipe.CreatedAt = createdAt.Format(timeLayout)
	recipe.UpdatedAt = updatedAt.Format(timeLayout)
	return recipe, nil
}
