// Package model はAPIで扱うドメインモデルを定義する。
package model

import "strconv"

// Cost は価格を表す型。API仕様に合わせ、レスポンスJSON上は文字列として出力する。
type Cost int

// MarshalJSON はCostをダブルクォートで囲んだ数値文字列としてエンコードする。
func (c Cost) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(strconv.Itoa(int(c)))), nil
}

// Recipe はrecipesテーブルの1レコードを表すレスポンス用モデル。
type Recipe struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	MakingTime  string `json:"making_time"`
	Serves      string `json:"serves"`
	Ingredients string `json:"ingredients"`
	Cost        Cost   `json:"cost"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// RecipeSummary はGET /recipes（一覧）で返す省略表現。created_at/updated_atを含まない。
type RecipeSummary struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	MakingTime  string `json:"making_time"`
	Serves      string `json:"serves"`
	Ingredients string `json:"ingredients"`
	Cost        Cost   `json:"cost"`
}

// ToSummary はRecipeを一覧表示用の省略表現に変換する。
func (r Recipe) ToSummary() RecipeSummary {
	return RecipeSummary{
		ID:          r.ID,
		Title:       r.Title,
		MakingTime:  r.MakingTime,
		Serves:      r.Serves,
		Ingredients: r.Ingredients,
		Cost:        r.Cost,
	}
}

// RecipePatchResult はPATCH /recipes/{id}で返す省略表現。idとタイムスタンプを含まない。
type RecipePatchResult struct {
	Title       string `json:"title"`
	MakingTime  string `json:"making_time"`
	Serves      string `json:"serves"`
	Ingredients string `json:"ingredients"`
	Cost        Cost   `json:"cost"`
}

// ToPatchResult はRecipeをPATCHレスポンス用の省略表現に変換する。
func (r Recipe) ToPatchResult() RecipePatchResult {
	return RecipePatchResult{
		Title:       r.Title,
		MakingTime:  r.MakingTime,
		Serves:      r.Serves,
		Ingredients: r.Ingredients,
		Cost:        r.Cost,
	}
}
