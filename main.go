package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Kate Mobile параметры
const (
	KATE_CLIENT_ID     = 2685278
	KATE_CLIENT_SECRET = "lxhD8OD7dMsqtXIm5IUY"
	KATE_API_VERSION   = 5.199
	KATE_USER_AGENT    = "KateMobileAndroid/51.1-442 (Android 14; SDK 34; arm64; Google; Google Pixel 7; ru)"
)

// Структура для ответа VK API
type VKResponse struct {
	Response json.RawMessage `json:"response"`
	Error    *struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	} `json:"error"`
}

// Прокси-сервер для обработки запросов к VK API
func vkProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем токен из запроса
	accessToken := r.URL.Query().Get("access_token")
	if accessToken == "" {
		// Пробуем получить из заголовка
		accessToken = r.Header.Get("X-Access-Token")
	}
	
	if accessToken == "" {
		http.Error(w, "Missing access_token", http.StatusBadRequest)
		return
	}
	
	// Определяем метод VK API
	method := strings.TrimPrefix(r.URL.Path, "/method/")
	if method == "" {
		method = r.URL.Query().Get("method")
	}
	
	if method == "" {
		http.Error(w, "Missing method", http.StatusBadRequest)
		return
	}
	
	log.Printf("Proxy request: method=%s, token=%s...", method, accessToken[:20])
	
	// Создаём запрос к VK API с Kate Mobile credentials
	vkURL := fmt.Sprintf("https://api.vk.com/method/%s", method)
	
	params := url.Values{}
	params.Set("v", fmt.Sprintf("%.3f", KATE_API_VERSION))
	params.Set("access_token", accessToken)
	
	// Добавляем остальные параметры из запроса
	for key, values := range r.URL.Query() {
		if key != "access_token" && key != "method" {
			params.Set(key, values[0])
		}
	}
	
	// Если есть тело запроса (для POST)
	var body io.Reader
	if r.Method == http.MethodPost {
		if r.Body != nil {
			body = r.Body
		}
	}
	
	fullURL := vkURL + "?" + params.Encode()
	log.Printf("Proxying to: %s", fullURL)
	
	req, err := http.NewRequest(r.Method, fullURL, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Устанавливаем User-Agent как в Kate Mobile
	req.Header.Set("User-Agent", KATE_USER_AGENT)
	req.Header.Set("X-Client-Id", strconv.Itoa(KATE_CLIENT_ID))
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Proxy error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	log.Printf("Proxy response status: %d, body length: %d", resp.StatusCode, len(respBody))
	
	// Проверяем на ошибку VK
	var vkResp VKResponse
	if err := json.Unmarshal(respBody, &vkResp); err == nil && vkResp.Error != nil {
		log.Printf("VK API error: %d - %s", vkResp.Error.ErrorCode, vkResp.Error.ErrorMsg)
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// Обработчик для получения URL трека
func getTrackURLHandler(w http.ResponseWriter, r *http.Request) {
	accessToken := r.URL.Query().Get("access_token")
	ownerID := r.URL.Query().Get("owner_id")
	trackID := r.URL.Query().Get("id")
	
	if accessToken == "" || ownerID == "" || trackID == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}
	
	// Запрашиваем информацию о треке через Kate Mobile API
	vkURL := fmt.Sprintf("https://api.vk.com/method/audio.getById?access_token=%s&audios=%s_%s&v=%.3f", 
		accessToken, ownerID, trackID, KATE_API_VERSION)
	
	req, err := http.NewRequest("GET", vkURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", KATE_USER_AGENT)
	
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	var result struct {
		Response []struct {
			URL string `json:"url"`
		} `json:"response"`
		Error *struct {
			ErrorMsg string `json:"error_msg"`
		} `json:"error"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	if result.Error != nil {
		http.Error(w, result.Error.ErrorMsg, http.StatusBadRequest)
		return
	}
	
	if len(result.Response) == 0 {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}
	
	trackURL := strings.ReplaceAll(result.Response[0].URL, "\\/", "/")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": trackURL})
}

func main() {
	port := "8081"
	
	log.Printf("==================================================")
	log.Printf("VK API PROXY (Kate Mobile compatible)")
	log.Printf("==================================================")
	log.Printf("✅ Proxy server starting on http://localhost:%s", port)
	log.Printf("🔑 Kate Client ID: %d", KATE_CLIENT_ID)
	log.Printf("📱 User-Agent: %s", KATE_USER_AGENT)
	log.Printf("==================================================")
	
	http.HandleFunc("/method/", vkProxyHandler)
	http.HandleFunc("/get-track-url", getTrackURLHandler)
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
