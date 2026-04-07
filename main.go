package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Track struct {
	ID       int    `json:"id"`
	OwnerID  int    `json:"owner_id"`
	Artist   string `json:"artist"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	URL      string `json:"url"`
}

type VKResponse struct {
	Response *struct {
		Count int     `json:"count"`
		Items []Track `json:"items"`
	} `json:"response"`
	Error *struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	} `json:"error"`
}

type DeleteResponse struct {
	Response int `json:"response"`
	Error    *struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	} `json:"error"`
}

type AddResponse struct {
	Response *struct {
		ID int `json:"id"`
	} `json:"response"`
	Error *struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	} `json:"error"`
}

type VKUserInfo struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Photo50   string `json:"photo_50"`
}

var trackURLCache = struct {
	sync.RWMutex
	data map[string]string
}{data: make(map[string]string)}

// Конфигурация OAuth
var (
	clientID     = "54529729"
	clientSecret = "" // Оставь пустым для Public приложения
	redirectURI  = "https://my-projets.onrender.com/"
)

func getTrackURL(token string, ownerID int, trackID int) string {
	cacheKey := fmt.Sprintf("%d_%d", ownerID, trackID)

	trackURLCache.RLock()
	if url, ok := trackURLCache.data[cacheKey]; ok {
		trackURLCache.RUnlock()
		return url
	}
	trackURLCache.RUnlock()

	client := &http.Client{Timeout: 10 * time.Second}
	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("audios", fmt.Sprintf("%d_%d", ownerID, trackID))

	apiURL := "https://api.vk.com/method/audio.getById?" + params.Encode()

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Response []struct {
			URL string `json:"url"`
		} `json:"response"`
		Error *struct {
			ErrorMsg string `json:"error_msg"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}

	if result.Error != nil || len(result.Response) == 0 {
		return ""
	}

	url := strings.ReplaceAll(result.Response[0].URL, "\\/", "/")

	trackURLCache.Lock()
	trackURLCache.data[cacheKey] = url
	trackURLCache.Unlock()

	return url
}

func searchTracks(token string, query string) ([]Track, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("q", query)
	params.Set("count", "30")

	apiURL := "https://api.vk.com/method/audio.search?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var vkResp VKResponse
	if err := json.Unmarshal(body, &vkResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга: %v", err)
	}

	if vkResp.Error != nil {
		return nil, fmt.Errorf("VK API ошибка: %s", vkResp.Error.ErrorMsg)
	}

	if vkResp.Response == nil {
		return []Track{}, nil
	}

	return vkResp.Response.Items, nil
}

func addTrack(token string, ownerID int, trackID int) error {
	client := &http.Client{Timeout: 10 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("owner_id", strconv.Itoa(ownerID))
	params.Set("audio_id", strconv.Itoa(trackID))

	apiURL := "https://api.vk.com/method/audio.add?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var addResp AddResponse
	if err := json.Unmarshal(body, &addResp); err != nil {
		return fmt.Errorf("ошибка парсинга: %v", err)
	}

	if addResp.Error != nil {
		return fmt.Errorf("VK API ошибка: %s", addResp.Error.ErrorMsg)
	}

	return nil
}

func getAllTracks(token string, ownerID int) ([]Track, error) {
	var allTracks []Track
	offset := 0
	count := 100

	client := &http.Client{Timeout: 15 * time.Second}

	for {
		params := url.Values{}
		params.Set("access_token", token)
		params.Set("v", "5.131")
		params.Set("count", strconv.Itoa(count))
		params.Set("offset", strconv.Itoa(offset))

		if ownerID != 0 {
			params.Set("owner_id", strconv.Itoa(ownerID))
		}

		apiURL := "https://api.vk.com/method/audio.get?" + params.Encode()

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return nil, err
		}

		var vkResp VKResponse
		if err := json.Unmarshal(body, &vkResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга: %v", err)
		}

		if vkResp.Error != nil {
			if vkResp.Error.ErrorCode == 5 {
				return nil, fmt.Errorf("unauthorized")
			}
			return nil, fmt.Errorf("VK API ошибка: %s", vkResp.Error.ErrorMsg)
		}

		if vkResp.Response == nil {
			break
		}

		allTracks = append(allTracks, vkResp.Response.Items...)

		if len(vkResp.Response.Items) < count {
			break
		}

		offset += count
		time.Sleep(50 * time.Millisecond)
	}

	return allTracks, nil
}

func deleteTrack(token string, ownerID int, trackID int) error {
	client := &http.Client{Timeout: 10 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("owner_id", strconv.Itoa(ownerID))
	params.Set("audio_id", strconv.Itoa(trackID))

	apiURL := "https://api.vk.com/method/audio.delete?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var deleteResp DeleteResponse
	if err := json.Unmarshal(body, &deleteResp); err != nil {
		return fmt.Errorf("ошибка парсинга: %v", err)
	}

	if deleteResp.Error != nil {
		return fmt.Errorf("VK API ошибка: %s", deleteResp.Error.ErrorMsg)
	}

	if deleteResp.Response != 1 {
		return fmt.Errorf("не удалось удалить трек")
	}

	return nil
}

func getUserInfo(token string) (*VKUserInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("fields", "photo_50")

	apiURL := "https://api.vk.com/method/users.get?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Response []VKUserInfo `json:"response"`
		Error    *struct {
			ErrorCode int    `json:"error_code"`
			ErrorMsg  string `json:"error_msg"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("VK API ошибка: %s", result.Error.ErrorMsg)
	}

	if len(result.Response) == 0 {
		return nil, fmt.Errorf("пользователь не найден")
	}

	return &result.Response[0], nil
}

func generateState() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Обработчик обмена кода на токен
func exchangeCodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Code     string `json:"code"`
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Обмениваем code на access_token через VK API
	tokenURL := fmt.Sprintf(
		"https://oauth.vk.com/access_token?client_id=%s&client_secret=%s&redirect_uri=%s&code=%s&device_id=%s",
		clientID, clientSecret, redirectURI, req.Code, req.DeviceID,
	)

	resp, err := http.Get(tokenURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		UserID      int    `json:"user_id"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse response"})
		return
	}

	if tokenResp.Error != "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": tokenResp.ErrorDesc})
		return
	}

	// Получаем информацию о пользователе
	userInfo, err := getUserInfo(tokenResp.AccessToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Сохраняем токен в куки
	http.SetCookie(w, &http.Cookie{
		Name:     "vk_token",
		Value:    tokenResp.AccessToken,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_user_id",
		Value:    strconv.Itoa(userInfo.ID),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_user_name",
		Value:    userInfo.FirstName + " " + userInfo.LastName,
		HttpOnly: false,
		Secure:   r.TLS != nil,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

// Страница входа с VK ID OneTap и ручным вводом токена
func loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=no">
    <title>VK Moosic — Вход</title>
    <link href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@20..48,100..700,0,1" />
    <script src="https://unpkg.com/@vkid/sdk@<3.0.0/dist-sdk/umd/index.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Roboto', sans-serif;
            min-height: 100vh;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .login-container { max-width: 450px; width: 100%; }
        .login-card {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 32px;
            padding: 40px 32px;
            box-shadow: 0 25px 50px -12px rgba(0,0,0,0.25);
            transition: transform 0.3s ease;
        }
        .login-card:hover { transform: translateY(-5px); }
        .logo { text-align: center; margin-bottom: 32px; }
        .logo-icon {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            width: 80px; height: 80px;
            border-radius: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 20px;
            box-shadow: 0 10px 25px -5px rgba(0,0,0,0.2);
        }
        .logo-icon .material-symbols-outlined { font-size: 48px; color: white; }
        .logo h1 {
            font-size: 28px;
            font-weight: 700;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 8px;
        }
        .logo p { color: #6b7280; font-size: 14px; }
        .vk-login-container {
            display: flex;
            justify-content: center;
            margin-bottom: 24px;
        }
        #vkIdOneTap {
            min-height: 60px;
            display: flex;
            justify-content: center;
            align-items: center;
        }
        .divider {
            text-align: center;
            margin: 20px 0;
            position: relative;
        }
        .divider::before {
            content: "";
            position: absolute;
            top: 50%;
            left: 0;
            right: 0;
            height: 1px;
            background: #e5e7eb;
        }
        .divider span {
            background: white;
            padding: 0 16px;
            position: relative;
            font-size: 13px;
            color: #9ca3af;
        }
        .token-input {
            width: 100%;
            padding: 16px;
            border: 2px solid #e5e7eb;
            border-radius: 16px;
            font-size: 14px;
            font-family: monospace;
            margin-bottom: 16px;
            outline: none;
            transition: all 0.2s;
        }
        .token-input:focus { border-color: #667eea; }
        .btn-token {
            width: 100%;
            background: #6b7280;
            color: white;
            border: none;
            padding: 14px;
            border-radius: 16px;
            font-size: 14px;
            cursor: pointer;
            transition: all 0.2s;
        }
        .btn-token:hover { background: #4b5563; transform: translateY(-2px); }
        .footer {
            text-align: center;
            margin-top: 24px;
            font-size: 12px;
            color: #9ca3af;
        }
        .loading-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0,0,0,0.7);
            display: none;
            align-items: center;
            justify-content: center;
            z-index: 1000;
        }
        .loading-overlay.active { display: flex; }
        .loading-spinner {
            width: 50px;
            height: 50px;
            border: 4px solid rgba(255,255,255,0.3);
            border-top-color: white;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        @media (max-width: 480px) { .login-card { padding: 32px 24px; } }
        @media (prefers-color-scheme: dark) {
            .login-card { background: rgba(31, 41, 55, 0.95); }
            .logo p { color: #9ca3af; }
            .divider span { background: #1f2937; color: #6b7280; }
            .token-input { background: #374151; border-color: #4b5563; color: white; }
            .token-input::placeholder { color: #9ca3af; }
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-card">
            <div class="logo">
                <div class="logo-icon">
                    <span class="material-symbols-outlined">music_note</span>
                </div>
                <h1>VK Moosic</h1>
                <p>Войдите, чтобы слушать музыку</p>
            </div>
            
            <div class="vk-login-container">
                <div id="vkIdOneTap"></div>
            </div>
            
            <div class="divider">
                <span>или введите токен вручную</span>
            </div>
            
            <form method="POST" action="/auth/set-token" id="tokenForm">
                <input type="text" name="token" class="token-input" placeholder="Введите access_token" autocomplete="off">
                <button type="submit" class="btn-token">Войти по токену</button>
            </form>
            
            <div class="footer">
                <span class="material-symbols-outlined" style="font-size: 14px; vertical-align: middle;">security</span>
                Токен хранится только в вашем браузере
            </div>
        </div>
    </div>
    
    <div class="loading-overlay" id="loadingOverlay">
        <div class="loading-spinner"></div>
    </div>

    <script>
        const loadingOverlay = document.getElementById('loadingOverlay');
        
        function showLoading() {
            loadingOverlay.classList.add('active');
        }
        
        function hideLoading() {
            loadingOverlay.classList.remove('active');
        }
        
        async function exchangeCodeForToken(code, deviceId) {
            showLoading();
            try {
                const response = await fetch('/auth/exchange-code', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ code: code, device_id: deviceId })
                });
                
                const data = await response.json();
                
                if (data.error) {
                    console.error('Ошибка:', data.error);
                    alert('Ошибка авторизации: ' + data.error);
                    hideLoading();
                    return;
                }
                
                if (data.success) {
                    window.location.href = '/';
                }
            } catch (err) {
                console.error('Ошибка:', err);
                alert('Ошибка соединения: ' + err.message);
                hideLoading();
            }
        }
        
        if (window.VKIDSDK) {
            const VKID = window.VKIDSDK;
            
            VKID.Config.init({
                app: 54529729,
                redirectUrl: 'https://my-projets.onrender.com/',
                responseMode: VKID.ConfigResponseMode.Callback,
                source: VKID.ConfigSource.LOWCODE,
                scope: 'audio,offline',
            });
            
            const oneTap = new VKID.OneTap();
            
            oneTap.render({
                container: document.getElementById('vkIdOneTap'),
                showAlternativeLogin: true
            })
            .on(VKID.WidgetEvents.ERROR, function(error) {
                console.error('VK ID Error:', error);
            })
            .on(VKID.OneTapInternalEvents.LOGIN_SUCCESS, function(payload) {
                const code = payload.code;
                const deviceId = payload.device_id;
                exchangeCodeForToken(code, deviceId);
            });
        } else {
            console.error('VKIDSDK not loaded');
            document.getElementById('vkIdOneTap').innerHTML = '<p style="color: red;">Ошибка загрузки VK ID</p>';
        }
        
        document.getElementById('tokenForm').addEventListener('submit', function() {
            showLoading();
        });
    </script>
</body>
</html>`

	w.Write([]byte(html))
}

func setTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	token := r.FormValue("token")
	if token == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userInfo, err := getUserInfo(token)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html>
			<head><title>Ошибка</title></head>
			<body style="font-family: Arial; text-align: center; padding: 50px;">
				<h2>❌ Ошибка авторизации</h2>
				<p>%s</p>
				<a href="/login">↺ Попробовать снова</a>
			</body>
			</html>
		`, err.Error())
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_token",
		Value:    token,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_user_id",
		Value:    strconv.Itoa(userInfo.ID),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_user_name",
		Value:    userInfo.FirstName + " " + userInfo.LastName,
		HttpOnly: false,
		Secure:   r.TLS != nil,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "vk_token", Value: "", Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: "vk_user_id", Value: "", Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: "vk_user_name", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenCookie, err := r.Cookie("vk_token")
		if err != nil || tokenCookie.Value == "" {
			if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/download" || r.URL.Path == "/get-track-url" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized", "redirect": "/login"})
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// API обработчики
func apiRecommendationsHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")

	client := &http.Client{Timeout: 15 * time.Second}
	params := url.Values{}
	params.Set("access_token", tokenCookie.Value)
	params.Set("v", "5.131")
	params.Set("count", "50")

	apiURL := "https://api.vk.com/method/audio.getRecommendations?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442")

	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var vkResp VKResponse
	if err := json.Unmarshal(body, &vkResp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Ошибка парсинга: %v", err)})
		return
	}

	if vkResp.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("VK API ошибка: %s", vkResp.Error.ErrorMsg)})
		return
	}

	if vkResp.Response == nil {
		json.NewEncoder(w).Encode([]Track{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vkResp.Response.Items)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "index.html")
}

func getTrackURLHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, err := r.Cookie("vk_token")
	if err != nil || tokenCookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	ownerID := r.URL.Query().Get("owner_id")
	trackID := r.URL.Query().Get("id")
	if ownerID == "" || trackID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing params"})
		return
	}

	ownerIDInt, _ := strconv.Atoi(ownerID)
	trackIDInt, _ := strconv.Atoi(trackID)

	url := getTrackURL(tokenCookie.Value, ownerIDInt, trackIDInt)
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

func downloadTrackHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, err := r.Cookie("vk_token")
	if err != nil || tokenCookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	trackID := r.URL.Query().Get("id")
	ownerID := r.URL.Query().Get("owner_id")
	if trackID == "" || ownerID == "" {
		http.Error(w, "Missing params", http.StatusBadRequest)
		return
	}

	ownerIDInt, _ := strconv.Atoi(ownerID)
	trackIDInt, _ := strconv.Atoi(trackID)

	trackURL := getTrackURL(tokenCookie.Value, ownerIDInt, trackIDInt)
	if trackURL == "" {
		http.Error(w, "Track URL not found", http.StatusNotFound)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	params := url.Values{}
	params.Set("access_token", tokenCookie.Value)
	params.Set("v", "5.131")
	params.Set("audios", ownerID+"_"+trackID)

	apiURL := "https://api.vk.com/method/audio.getById?" + params.Encode()

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442")

	resp, _ := client.Do(req)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Response []struct {
			Artist string `json:"artist"`
			Title  string `json:"title"`
		} `json:"response"`
	}
	json.Unmarshal(body, &result)

	artist := "Unknown"
	title := "Track"
	if len(result.Response) > 0 {
		artist = result.Response[0].Artist
		title = result.Response[0].Title
	}

	downloadResp, err := http.Get(trackURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer downloadResp.Body.Close()

	filename := fmt.Sprintf("%s - %s.mp3", artist, title)
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")
	filename = strings.ReplaceAll(filename, ":", "-")
	filename = strings.ReplaceAll(filename, "\"", "-")
	filename = strings.ReplaceAll(filename, "?", "-")
	filename = strings.ReplaceAll(filename, "*", "-")

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(filename)))

	io.Copy(w, downloadResp.Body)
}

func apiTracksHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")
	userIDCookie, _ := r.Cookie("vk_user_id")
	userID, _ := strconv.Atoi(userIDCookie.Value)

	tracks, err := getAllTracks(tokenCookie.Value, userID)
	if err != nil {
		if err.Error() == "unauthorized" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized", "redirect": "/login"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

func apiSearchHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")
	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode([]Track{})
		return
	}

	tracks, err := searchTracks(tokenCookie.Value, query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

func apiAddTrackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tokenCookie, _ := r.Cookie("vk_token")
	userIDCookie, _ := r.Cookie("vk_user_id")
	userID, _ := strconv.Atoi(userIDCookie.Value)

	var req struct {
		TrackID int `json:"track_id"`
		OwnerID int `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	err := addTrack(tokenCookie.Value, userID, req.TrackID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

func apiDeleteTrackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tokenCookie, _ := r.Cookie("vk_token")
	userIDCookie, _ := r.Cookie("vk_user_id")
	userID, _ := strconv.Atoi(userIDCookie.Value)

	var req struct {
		TrackID int `json:"track_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	err := deleteTrack(tokenCookie.Value, userID, req.TrackID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

func apiUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	userNameCookie, _ := r.Cookie("vk_user_name")
	name := "Пользователь"
	if userNameCookie != nil && userNameCookie.Value != "" {
		name = userNameCookie.Value
	}
	json.NewEncoder(w).Encode(map[string]string{"name": name})
}

func main() {
	// Получаем настройки из переменных окружения (для Render)
	if os.Getenv("VK_CLIENT_ID") != "" {
		clientID = os.Getenv("VK_CLIENT_ID")
	}
	if os.Getenv("VK_CLIENT_SECRET") != "" {
		clientSecret = os.Getenv("VK_CLIENT_SECRET")
	}
	if os.Getenv("VK_REDIRECT_URI") != "" {
		redirectURI = os.Getenv("VK_REDIRECT_URI")
	}

	// Обработчики
	http.HandleFunc("/", authMiddleware(indexHandler))
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/auth/exchange-code", exchangeCodeHandler)
	http.HandleFunc("/auth/set-token", setTokenHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/download", authMiddleware(downloadTrackHandler))
	http.HandleFunc("/get-track-url", authMiddleware(getTrackURLHandler))
	http.HandleFunc("/api/tracks", authMiddleware(apiTracksHandler))
	http.HandleFunc("/api/search", authMiddleware(apiSearchHandler))
	http.HandleFunc("/api/add", authMiddleware(apiAddTrackHandler))
	http.HandleFunc("/api/delete", authMiddleware(apiDeleteTrackHandler))
	http.HandleFunc("/api/user", authMiddleware(apiUserInfoHandler))
	http.HandleFunc("/api/recommendations", authMiddleware(apiRecommendationsHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("==================================================")
	fmt.Println("🎵 VK MOOSIC WEB PLAYER")
	fmt.Println("==================================================")
	fmt.Printf("\n✅ Сервер запущен: http://localhost:%s\n", port)
	fmt.Println("🔐 Доступны два способа входа:")
	fmt.Println("   • Через VK ID (OneTap)")
	fmt.Println("   • По токену вручную")
	fmt.Println("\n⚠️ ВНИМАНИЕ: Удаление треков реально удаляет их из ВКонтакте!")
	fmt.Println("==================================================")

	http.ListenAndServe(":"+port, nil)
}
