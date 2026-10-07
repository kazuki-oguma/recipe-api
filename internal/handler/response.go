// Package handler はrecipesリソースに対するHTTPハンドラを提供する。
package handler

import (
	"encoding/json"
	"net/http"
)

// writeJSON はstatusとpayloadをJSONレスポンスとして書き込む共通ヘルパー。
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	// レスポンス書き込み後のエラーはクライアントへ通知できないためログ目的以外では無視する。
	_ = json.NewEncoder(w).Encode(payload)
}
