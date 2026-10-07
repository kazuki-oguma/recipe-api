// recipe-api はレシピ情報をCRUD操作するREST APIサーバーのエントリポイント。
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"recipe-api/internal/handler"
	"recipe-api/internal/repository"
)

func main() {
	db, err := newDB()
	if err != nil {
		log.Fatalf("データベース接続に失敗しました: %v", err)
	}
	defer db.Close()

	repo := repository.NewRecipeRepository(db)
	h := handler.NewRecipeHandler(repo)

	mux := newRouter(h)

	addr := ":" + getEnv("PORT", "8080")
	log.Printf("recipe-api を %s で起動します", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}

// newRouter はrecipesリソースに対するルーティングを構築する。
// パターンに一致しないパスはnet/http標準の挙動により404を返す。
func newRouter(h *handler.RecipeHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /recipes", h.CreateRecipe)
	mux.HandleFunc("GET /recipes", h.ListRecipes)
	mux.HandleFunc("GET /recipes/{id}", h.GetRecipe)
	mux.HandleFunc("PATCH /recipes/{id}", h.UpdateRecipe)
	mux.HandleFunc("DELETE /recipes/{id}", h.DeleteRecipe)
	return mux
}

// newDB は環境変数からPostgreSQLへの接続を確立する。
func newDB() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		getEnv("DB_HOST", "127.0.0.1"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", ""),
		getEnv("DB_NAME", "recipes_db"),
		getEnv("DB_SSLMODE", "disable"),
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
