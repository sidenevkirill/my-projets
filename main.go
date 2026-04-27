package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	KATE_API_VERSION = "5.199"
	KATE_USER_AGENT  = "KateMobileAndroid/51.1-442 (Android 14; SDK 34; arm64; Google; Google Pixel 7; ru)"
)

func main() {
	port := "8080"

	// Обработчик для всех методов VK API
	http.HandleFunc("/method/", func(w http.ResponseWriter, r *http.Request) {
		// Получаем метод из URL (например, /method/audio.get -> audio.get)
		method := strings.TrimPrefix(r.URL.Path, "/method/")

		// Если метод пустой, пробуем взять из query параметра
		if method == "" {
			method = r.URL.Query().Get("method")
		}

		if method == "" {
			http.Error(w, "Missing method", http.StatusBadRequest)
			return
		}

		// Получаем access_token
		accessToken := r.URL.Query().Get("access_token")
		if accessToken == "" {
			http.Error(w, "Missing access_token", http.StatusBadRequest)
			return
		}

		log.Printf("Proxying method: %s", method)

		// Строим запрос к VK API
		vkURL := "https://api.vk.com/method/" + method

		params := url.Values{}
		params.Set("v", KATE_API_VERSION)
		params.Set("access_token", accessToken)

		// Копируем остальные параметры из запроса
		for key, values := range r.URL.Query() {
			if key != "access_token" && key != "method" {
				params.Set(key, values[0])
			}
		}

		// Если есть тело запроса (для POST)
		var body io.Reader
		if r.Method == http.MethodPost && r.Body != nil {
			body = r.Body
		}

		fullURL := vkURL + "?" + params.Encode()
		log.Printf("Proxying to: %s", fullURL)

		req, err := http.NewRequest(r.Method, fullURL, body)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Устанавливаем User-Agent как в Kate Mobile
		req.Header.Set("User-Agent", KATE_USER_AGENT)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error making request: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Response status: %d, body length: %d", resp.StatusCode, len(respBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Корневой эндпоинт для проверки
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "running",
			"message": "VK API Proxy is working",
		})
	})

	log.Printf("==================================================")
	log.Printf("VK API PROXY (Kate Mobile compatible)")
	log.Printf("==================================================")
	log.Printf("✅ Proxy server starting on http://localhost:%s", port)
	log.Printf("🔑 Using Kate Mobile API v%s", KATE_API_VERSION)
	log.Printf("==================================================")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
