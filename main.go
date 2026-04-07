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

type Playlist struct {
	ID        int    `json:"id"`
	OwnerID   int    `json:"owner_id"`
	Title     string `json:"title"`
	Count     int    `json:"count"`
	Photo     string `json:"photo"`
	AccessKey string `json:"access_key"`
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

// Кэш для URL треков
var trackURLCache = struct {
	sync.RWMutex
	data map[string]string
}{data: make(map[string]string)}

// Получение ссылки на трек с кэшированием
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

// Получение URL для воспроизведения (с кэшем)
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

// Скачивание трека
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

	// Получаем информацию о треке
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

	// Скачиваем файл
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

// Получение списка плейлистов пользователя
// Получение треков плейлиста
func apiPlaylistTracksHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")
	// userIDCookie временно не используется, но может понадобиться позже
	// userIDCookie, _ := r.Cookie("vk_user_id")

	// Извлекаем параметры из URL
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid playlist ID"})
		return
	}
	
	playlistID := parts[3]
	ownerID := parts[4]

	client := &http.Client{Timeout: 15 * time.Second}
	params := url.Values{}
	params.Set("access_token", tokenCookie.Value)
	params.Set("v", "5.131")
	params.Set("owner_id", ownerID)
	params.Set("playlist_id", playlistID)

	apiURL := "https://api.vk.com/method/audio.getPlaylistById?" + params.Encode()

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

	fmt.Printf("Playlist tracks API response: %s\n", string(body))

	var result struct {
		Response struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Audios []struct {
				ID       int    `json:"id"`
				OwnerID  int    `json:"owner_id"`
				Artist   string `json:"artist"`
				Title    string `json:"title"`
				Duration int    `json:"duration"`
				URL      string `json:"url"`
			} `json:"audios"`
		} `json:"response"`
		Error *struct {
			ErrorCode int    `json:"error_code"`
			ErrorMsg  string `json:"error_msg"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Ошибка парсинга: %v", err)})
		return
	}

	if result.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("VK API ошибка: %s (код %d)", result.Error.ErrorMsg, result.Error.ErrorCode)})
		return
	}

	tracks := make([]Track, 0)
	for _, item := range result.Response.Audios {
		// Нормализуем URL если есть
		url := strings.ReplaceAll(item.URL, "\\/", "/")
		tracks = append(tracks, Track{
			ID:       item.ID,
			OwnerID:  item.OwnerID,
			Artist:   item.Artist,
			Title:    item.Title,
			Duration: item.Duration,
			URL:      url,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

// Получение треков плейлиста
func apiPlaylistTracksHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")
	userIDCookie, _ := r.Cookie("vk_user_id")

	// Извлекаем параметры из URL
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid playlist ID"})
		return
	}
	
	playlistID := parts[3]
	ownerID := parts[4]

	client := &http.Client{Timeout: 15 * time.Second}
	params := url.Values{}
	params.Set("access_token", tokenCookie.Value)
	params.Set("v", "5.131")
	params.Set("owner_id", ownerID)
	params.Set("playlist_id", playlistID)

	apiURL := "https://api.vk.com/method/audio.getPlaylistById?" + params.Encode()

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

	fmt.Printf("Playlist tracks API response: %s\n", string(body))

	var result struct {
		Response struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Audios []struct {
				ID       int    `json:"id"`
				OwnerID  int    `json:"owner_id"`
				Artist   string `json:"artist"`
				Title    string `json:"title"`
				Duration int    `json:"duration"`
				URL      string `json:"url"`
			} `json:"audios"`
		} `json:"response"`
		Error *struct {
			ErrorCode int    `json:"error_code"`
			ErrorMsg  string `json:"error_msg"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Ошибка парсинга: %v", err)})
		return
	}

	if result.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("VK API ошибка: %s (код %d)", result.Error.ErrorMsg, result.Error.ErrorCode)})
		return
	}

	tracks := make([]Track, 0)
	for _, item := range result.Response.Audios {
		// Нормализуем URL если есть
		url := strings.ReplaceAll(item.URL, "\\/", "/")
		tracks = append(tracks, Track{
			ID:       item.ID,
			OwnerID:  item.OwnerID,
			Artist:   item.Artist,
			Title:    item.Title,
			Duration: item.Duration,
			URL:      url,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

// Обработчики
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "index.html")
}

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
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Roboto', -apple-system, BlinkMacSystemFont, sans-serif;
            min-height: 100vh;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        
        .login-container {
            max-width: 450px;
            width: 100%;
        }
        
        .login-card {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 32px;
            padding: 40px 32px;
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
            transition: transform 0.3s ease;
        }
        
        .login-card:hover {
            transform: translateY(-5px);
        }
        
        .logo {
            text-align: center;
            margin-bottom: 32px;
        }
        
        .logo-icon {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            width: 80px;
            height: 80px;
            border-radius: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 20px;
            box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.2);
        }
        
        .logo-icon .material-symbols-outlined {
            font-size: 48px;
            color: white;
        }
        
        .logo h1 {
            font-size: 28px;
            font-weight: 700;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 8px;
        }
        
        .logo p {
            color: #6b7280;
            font-size: 14px;
        }
        
        .info-box {
            background: #f3f4f6;
            border-radius: 16px;
            padding: 16px;
            margin-bottom: 24px;
            display: flex;
            align-items: center;
            gap: 12px;
        }
        
        .info-box .material-symbols-outlined {
            color: #667eea;
            font-size: 24px;
        }
        
        .info-box-text {
            flex: 1;
            font-size: 13px;
            color: #4b5563;
            line-height: 1.4;
        }
        
        .info-box-text a {
            color: #667eea;
            text-decoration: none;
            font-weight: 500;
        }
        
        .info-box-text a:hover {
            text-decoration: underline;
        }
        
        .form-group {
            margin-bottom: 24px;
        }
        
        .input-wrapper {
            display: flex;
            align-items: center;
            background: #f9fafb;
            border: 2px solid #e5e7eb;
            border-radius: 16px;
            padding: 4px 16px;
            transition: all 0.2s;
        }
        
        .input-wrapper:focus-within {
            border-color: #667eea;
            background: white;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }
        
        .input-wrapper .material-symbols-outlined {
            color: #9ca3af;
            font-size: 20px;
            margin-right: 12px;
        }
        
        .input-wrapper input {
            flex: 1;
            border: none;
            background: none;
            padding: 16px 0;
            font-size: 16px;
            outline: none;
            font-family: 'Roboto', monospace;
            color: #1f2937;
        }
        
        .input-wrapper input::placeholder {
            color: #9ca3af;
            font-family: 'Roboto', sans-serif;
        }
        
        .token-hint {
            margin-top: 8px;
            font-size: 12px;
            color: #6b7280;
            padding-left: 12px;
        }
        
        .btn-login {
            width: 100%;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            padding: 16px;
            border-radius: 16px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }
        
        .btn-login:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 25px -5px rgba(102, 126, 234, 0.4);
        }
        
        .btn-login:active {
            transform: translateY(0);
        }
        
        .footer {
            text-align: center;
            margin-top: 24px;
            font-size: 12px;
            color: #9ca3af;
        }
        
        @media (max-width: 480px) {
            .login-card {
                padding: 32px 24px;
            }
            .logo h1 {
                font-size: 24px;
            }
        }
        
        @media (prefers-color-scheme: dark) {
            .login-card {
                background: rgba(31, 41, 55, 0.95);
            }
            .logo p {
                color: #9ca3af;
            }
            .info-box {
                background: #374151;
            }
            .info-box-text {
                color: #d1d5db;
            }
            .input-wrapper {
                background: #374151;
                border-color: #4b5563;
            }
            .input-wrapper input {
                color: #f3f4f6;
            }
            .token-hint {
                color: #9ca3af;
            }
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
            
            <div class="info-box">
                <span class="material-symbols-outlined">info</span>
                <div class="info-box-text">
                    Для входа нужен токен доступа ВКонтакте.<br>
                    <a href="#" onclick="showInstructions(); return false;">Как получить токен?</a>
                </div>
            </div>
            
            <form method="POST" action="/auth/set-token">
                <div class="form-group">
                    <div class="input-wrapper">
                        <span class="material-symbols-outlined">key</span>
                        <input type="text" name="token" id="tokenInput" placeholder="Введите access_token" required autocomplete="off">
                    </div>
                    <div class="token-hint">
                        Токен начинается с "vk1.a."
                    </div>
                </div>
                
                <button type="submit" class="btn-login">
                    <span class="material-symbols-outlined">login</span>
                    <span>Войти</span>
                </button>
            </form>
            
            <div class="footer">
                <span class="material-symbols-outlined" style="font-size: 14px; vertical-align: middle;">security</span>
                Токен хранится только в вашем браузере
            </div>
        </div>
    </div>
    
    <script>
        function showInstructions() {
            alert("Как получить токен ВКонтакте:\n\n1. Перейдите на сайт: vkhost.github.io\n2. Выберите приложение 'Kate Mobile' или 'VK Music'\n3. Отметьте права доступа: Аудиозаписи (audio)\n4. Нажмите 'Получить токен'\n5. Скопируйте access_token (начинается с vk1.a.)\n6. Вставьте его в поле выше");
        }
        
        const tokenInput = document.getElementById('tokenInput');
        tokenInput.addEventListener('paste', function(e) {
            setTimeout(function() {
                var value = tokenInput.value.trim();
                if (value && !value.startsWith('vk1.a.')) {
                    console.warn('Похоже, это не токен VK');
                }
            }, 100);
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
	http.HandleFunc("/", authMiddleware(indexHandler))
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/auth/set-token", setTokenHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/download", authMiddleware(downloadTrackHandler))
	http.HandleFunc("/get-track-url", authMiddleware(getTrackURLHandler))
	http.HandleFunc("/api/tracks", authMiddleware(apiTracksHandler))
	http.HandleFunc("/api/search", authMiddleware(apiSearchHandler))
	http.HandleFunc("/api/add", authMiddleware(apiAddTrackHandler))
	http.HandleFunc("/api/delete", authMiddleware(apiDeleteTrackHandler))
	http.HandleFunc("/api/user", authMiddleware(apiUserInfoHandler))
	http.HandleFunc("/api/playlists", authMiddleware(apiPlaylistsHandler))
	http.HandleFunc("/api/playlist/", authMiddleware(apiPlaylistTracksHandler))

	port := "8080"
	fmt.Println("==================================================")
	fmt.Println("🎵 VK MUSIC PLAYER")
	fmt.Println("==================================================")
	fmt.Printf("\n✅ Сервер запущен: http://localhost:%s\n", port)
	fmt.Println("🌐 Открой в браузере: http://localhost:8080")
	fmt.Println("🔐 Потребуется ввести токен ВК при первом входе")
	fmt.Println("\n📥 Треки можно скачать через кнопку меню (три точки)")
	fmt.Println("📀 Доступны плейлисты пользователя")
	fmt.Println("⚠️ Удаление треков реально удаляет их из ВКонтакте!")
	fmt.Println("📌 Нажми Ctrl+C для остановки")
	fmt.Println("==================================================")

	http.ListenAndServe(":"+port, nil)
}
