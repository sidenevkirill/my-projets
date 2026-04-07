package main

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strconv"
    "strings"
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

// VK OAuth структуры
type VKAccessTokenResponse struct {
    AccessToken string `json:"access_token"`
    UserID      int    `json:"user_id"`
    Email       string `json:"email"`
    Error       string `json:"error"`
    ErrorDesc   string `json:"error_description"`
}

type VKUserInfo struct {
    ID        int    `json:"id"`
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
    Photo50   string `json:"photo_50"`
}

func searchTracks(token string, query string) ([]Track, error) {
    client := &http.Client{Timeout: 30 * time.Second}

    params := url.Values{}
    params.Set("access_token", token)
    params.Set("v", "5.131")
    params.Set("q", query)
    params.Set("count", "50")

    apiURL := "https://api.vk.com/method/audio.search?" + params.Encode()

    req, err := http.NewRequest("GET", apiURL, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442 (Android 11; SDK 30; arm64-v8a; ru)")

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
    client := &http.Client{Timeout: 30 * time.Second}

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
    req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442 (Android 11; SDK 30; arm64-v8a; ru)")

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
        return fmt.Errorf("VK API ошибка: %s (код %d)", addResp.Error.ErrorMsg, addResp.Error.ErrorCode)
    }

    return nil
}

func getAllTracks(token string, ownerID int) ([]Track, int, error) {
    var allTracks []Track
    offset := 0
    count := 100
    var totalCount int

    client := &http.Client{Timeout: 30 * time.Second}

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
            return nil, 0, err
        }
        req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442 (Android 11; SDK 30; arm64-v8a; ru)")

        resp, err := client.Do(req)
        if err != nil {
            return nil, 0, err
        }

        body, err := io.ReadAll(resp.Body)
        resp.Body.Close()

        if err != nil {
            return nil, 0, err
        }

        var vkResp VKResponse
        if err := json.Unmarshal(body, &vkResp); err != nil {
            return nil, 0, fmt.Errorf("ошибка парсинга: %v", err)
        }

        if vkResp.Error != nil {
            return nil, 0, fmt.Errorf("VK API ошибка: %s", vkResp.Error.ErrorMsg)
        }

        if vkResp.Response == nil {
            return nil, 0, fmt.Errorf("пустой ответ")
        }

        if totalCount == 0 {
            totalCount = vkResp.Response.Count
        }

        allTracks = append(allTracks, vkResp.Response.Items...)

        fmt.Printf("Загружено %d из %d треков\n", len(allTracks), totalCount)

        if len(vkResp.Response.Items) < count {
            break
        }

        offset += count
        if offset >= totalCount {
            break
        }

        if offset > 5000 {
            break
        }

        time.Sleep(100 * time.Millisecond)
    }

    return allTracks, totalCount, nil
}

func deleteTrack(token string, ownerID int, trackID int) error {
    client := &http.Client{Timeout: 30 * time.Second}

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
    req.Header.Set("User-Agent", "KateMobileAndroid/51.1-442 (Android 11; SDK 30; arm64-v8a; ru)")

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
        return fmt.Errorf("VK API ошибка: %s (код %d)", deleteResp.Error.ErrorMsg, deleteResp.Error.ErrorCode)
    }

    if deleteResp.Response != 1 {
        return fmt.Errorf("не удалось удалить трек, ответ: %d", deleteResp.Response)
    }

    return nil
}

// Получение информации о пользователе
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

// Генерация случайной строки для state
func generateState() string {
    bytes := make([]byte, 16)
    rand.Read(bytes)
    return hex.EncodeToString(bytes)
}

// Обработчики
func indexHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    http.ServeFile(w, r, "index.html")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
    state := generateState()
    // Сохраняем state в сессию (через куку)
    http.SetCookie(w, &http.Cookie{
        Name:     "vk_auth_state",
        Value:    state,
        HttpOnly: true,
        Secure:   r.TLS != nil,
        Path:     "/",
        MaxAge:   600, // 10 минут
    })
    
    // Строим URL для OAuth
    // Нужно зарегистрировать приложение в VK и получить client_id
    // Пока используем прямой токен, но для полноценной OAuth нужно:
    // redirectURI := fmt.Sprintf("http://%s/auth/callback", r.Host)
    // authURL := fmt.Sprintf("https://oauth.vk.com/authorize?client_id=%s&display=page&redirect_uri=%s&scope=audio&response_type=code&v=5.131&state=%s", clientID, redirectURI, state)
    
    // Временно возвращаем инструкцию
    w.Header().Set("Content-Type", "text/html")
    fmt.Fprintf(w, `
        <!DOCTYPE html>
        <html>
        <head><title>Авторизация ВК</title></head>
        <body style="font-family: Arial; text-align: center; padding: 50px;">
            <h2>🔐 Авторизация ВКонтакте</h2>
            <p>Для использования плеера введите ваш токен доступа ВК:</p>
            <form method="POST" action="/auth/set-token">
                <input type="text" name="token" placeholder="Введите токен VK" style="width: 300px; padding: 10px; margin: 10px;" required>
                <br>
                <button type="submit" style="padding: 10px 30px; background: #4b7cad; color: white; border: none; border-radius: 5px; cursor: pointer;">Войти</button>
            </form>
            <p style="margin-top: 30px; font-size: 12px; color: gray;">
                <a href="#" onclick="alert('Как получить токен:\\n1. Перейдите на vkhost.github.io\\n2. Выберите нужные права (аудио)\\n3. Скопируйте access_token')">❓ Как получить токен?</a>
            </p>
        </body>
        </html>
    `)
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
    
    // Проверяем токен
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
    
    // Сохраняем токен и информацию о пользователе в сессию
    http.SetCookie(w, &http.Cookie{
        Name:     "vk_token",
        Value:    token,
        HttpOnly: true,
        Secure:   r.TLS != nil,
        Path:     "/",
        MaxAge:   86400 * 30, // 30 дней
    })
    
    // Сохраняем ID пользователя
    http.SetCookie(w, &http.Cookie{
        Name:     "vk_user_id",
        Value:    strconv.Itoa(userInfo.ID),
        HttpOnly: true,
        Secure:   r.TLS != nil,
        Path:     "/",
        MaxAge:   86400 * 30,
    })
    
    // Сохраняем имя пользователя (не обязательно, но приятно)
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
    http.SetCookie(w, &http.Cookie{
        Name:     "vk_token",
        Value:    "",
        HttpOnly: true,
        Path:     "/",
        MaxAge:   -1,
    })
    http.SetCookie(w, &http.Cookie{
        Name:     "vk_user_id",
        Value:    "",
        HttpOnly: true,
        Path:     "/",
        MaxAge:   -1,
    })
    http.SetCookie(w, &http.Cookie{
        Name:     "vk_user_name",
        Value:    "",
        Path:     "/",
        MaxAge:   -1,
    })
    http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        tokenCookie, err := r.Cookie("vk_token")
        if err != nil || tokenCookie.Value == "" {
            // Если запрос к API, возвращаем ошибку
            if strings.HasPrefix(r.URL.Path, "/api/") {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusUnauthorized)
                json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized", "redirect": "/login"})
                return
            }
            // Для остальных запросов редирект на логин
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }
        
        // Проверяем валидность токена (опционально, для уверенности)
        _, err = getUserInfo(tokenCookie.Value)
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }
        
        next(w, r)
    }
}

func apiTracksHandler(w http.ResponseWriter, r *http.Request) {
    tokenCookie, _ := r.Cookie("vk_token")
    userIDCookie, _ := r.Cookie("vk_user_id")
    userID, _ := strconv.Atoi(userIDCookie.Value)
    
    tracks, _, err := getAllTracks(tokenCookie.Value, userID)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
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
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode([]Track{})
        return
    }

    tracks, err := searchTracks(tokenCookie.Value, query)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
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
        json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
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

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"success": "true", "message": "Трек добавлен в ВКонтакте"})
}

func apiDeleteTrackHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete && r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
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

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"success": "true", "message": "Трек удалён из ВКонтакте"})
}

func apiUserInfoHandler(w http.ResponseWriter, r *http.Request) {
    userNameCookie, _ := r.Cookie("vk_user_name")
    userInfo := map[string]string{
        "name": userNameCookie.Value,
    }
    if userNameCookie == nil || userNameCookie.Value == "" {
        userInfo["name"] = "Пользователь"
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(userInfo)
}

func main() {
    // Обёрнутые в middleware обработчики API
    http.HandleFunc("/", authMiddleware(indexHandler))
    http.HandleFunc("/login", loginHandler)
    http.HandleFunc("/auth/set-token", setTokenHandler)
    http.HandleFunc("/logout", logoutHandler)
    http.HandleFunc("/api/tracks", authMiddleware(apiTracksHandler))
    http.HandleFunc("/api/search", authMiddleware(apiSearchHandler))
    http.HandleFunc("/api/add", authMiddleware(apiAddTrackHandler))
    http.HandleFunc("/api/delete", authMiddleware(apiDeleteTrackHandler))
    http.HandleFunc("/api/user", authMiddleware(apiUserInfoHandler))

    port := "8080"
    fmt.Println("==================================================")
    fmt.Println("VK MUSIC PLAYER")
    fmt.Println("==================================================")
    fmt.Printf("\n✅ Сервер запущен: http://localhost:%s\n", port)
    fmt.Println("🌐 Открой в браузере: http://localhost:8080")
    fmt.Println("🔐 Потребуется ввести токен ВК при первом входе")
    fmt.Println("\n⚠️ ВНИМАНИЕ: Удаление треков реально удаляет их из ВКонтакте!")
    fmt.Println("📌 Нажми Ctrl+C для остановки")
    fmt.Println("==================================================")

    http.ListenAndServe(":"+port, nil)
}
