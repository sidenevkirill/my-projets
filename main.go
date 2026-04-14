package main

import (
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
	ID         int    `json:"id"`
	OwnerID    int    `json:"owner_id"`
	Title      string `json:"title"`
	Count      int    `json:"count"`
	Photo      struct {
		ID        int    `json:"id"`
		Photo75   string `json:"photo_75"`
		Photo130  string `json:"photo_130"`
		Photo270  string `json:"photo_270"`
		Photo300  string `json:"photo_300"`
		Photo600  string `json:"photo_600"`
		Photo1200 string `json:"photo_1200"`
	} `json:"photo"`
	AccessKey  string `json:"access_key"`
	IsFollowed bool   `json:"is_followed"`
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
	Photo100  string `json:"photo_100"`
}

type ConversationsResponse struct {
	Response struct {
		Count int `json:"count"`
		Items []struct {
			Conversation struct {
				Peer struct {
					ID int `json:"id"`
				} `json:"peer"`
				UnreadCount int `json:"unread_count"`
			} `json:"conversation"`
			LastMessage struct {
				ID     int    `json:"id"`
				Date   int64  `json:"date"`
				FromID int    `json:"from_id"`
				Text   string `json:"text"`
				Out    int    `json:"out"`
			} `json:"last_message"`
		} `json:"items"`
		Profiles []struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"profiles"`
	} `json:"response"`
	Error *struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	} `json:"error"`
}

type MessagesResponse struct {
	Response struct {
		Count int `json:"count"`
		Items []struct {
			ID     int    `json:"id"`
			Date   int64  `json:"date"`
			FromID int    `json:"from_id"`
			PeerID int    `json:"peer_id"`
			Text   string `json:"text"`
			Out    int    `json:"out"`
		} `json:"items"`
		Profiles []struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"profiles"`
	} `json:"response"`
	Error *struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	} `json:"error"`
}

type Friend struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Photo100  string `json:"photo_100"`
	Online    int    `json:"online"`
}

type Group struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
	Photo100   string `json:"photo_100"`
	IsClosed   int    `json:"is_closed"`
	Type       string `json:"type"`
}

type FriendsResponse struct {
	Response struct {
		Count int      `json:"count"`
		Items []Friend `json:"items"`
	} `json:"response"`
	Error *struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	} `json:"error"`
}

type GroupsResponse struct {
	Response struct {
		Count int     `json:"count"`
		Items []Group `json:"items"`
	} `json:"response"`
	Error *struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	} `json:"error"`
}

type SetOnlineResponse struct {
	Response int `json:"response"`
	Error    *struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	} `json:"error"`
}

// Константы для прокси
const (
	API_PROXY   = "https://vk-api-proxy.xtrafrancyz.net/method/"
	OAUTH_PROXY = "https://vk-oauth-proxy.xtrafrancyz.net/"
)

var useProxy = false

var trackURLCache = struct {
	sync.RWMutex
	data map[string]string
}{data: make(map[string]string)}

func getAPIBaseURL() string {
	if useProxy {
		return API_PROXY
	}
	return "https://api.vk.com/method/"
}

func getOAuthBaseURL() string {
	if useProxy {
		return OAUTH_PROXY
	}
	return "https://oauth.vk.com/"
}

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

	apiURL := getAPIBaseURL() + "audio.getById?" + params.Encode()

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
	params.Set("count", "50")

	apiURL := getAPIBaseURL() + "audio.search?" + params.Encode()

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

	apiURL := getAPIBaseURL() + "audio.add?" + params.Encode()

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

		apiURL := getAPIBaseURL() + "audio.get?" + params.Encode()

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

	apiURL := getAPIBaseURL() + "audio.delete?" + params.Encode()

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
	params.Set("fields", "photo_50,photo_100")

	apiURL := getAPIBaseURL() + "users.get?" + params.Encode()

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

// ========== Функции для работы с плейлистами ==========

func getPlaylists(token string, ownerID int) ([]Playlist, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("owner_id", strconv.Itoa(ownerID))
	params.Set("count", "100")
	params.Set("extended", "1")

	apiURL := getAPIBaseURL() + "audio.getPlaylists?" + params.Encode()

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

	var rawResult map[string]interface{}
	if err := json.Unmarshal(body, &rawResult); err != nil {
		return nil, fmt.Errorf("ошибка парсинга: %v", err)
	}

	if errObj, ok := rawResult["error"]; ok {
		errMap := errObj.(map[string]interface{})
		return nil, fmt.Errorf("VK API ошибка: %v", errMap["error_msg"])
	}

	response, ok := rawResult["response"].(map[string]interface{})
	if !ok {
		return []Playlist{}, nil
	}

	var items []interface{}
	if itemsRaw, ok := response["items"]; ok {
		items = itemsRaw.([]interface{})
	} else if playlistsRaw, ok := response["playlists"]; ok {
		items = playlistsRaw.([]interface{})
	} else {
		return []Playlist{}, nil
	}

	playlists := make([]Playlist, 0, len(items))
	for _, itemRaw := range items {
		item := itemRaw.(map[string]interface{})

		playlist := Playlist{}

		if id, ok := item["id"].(float64); ok {
			playlist.ID = int(id)
		}
		if ownerID, ok := item["owner_id"].(float64); ok {
			playlist.OwnerID = int(ownerID)
		}
		if title, ok := item["title"].(string); ok {
			playlist.Title = title
		}
		if count, ok := item["count"].(float64); ok {
			playlist.Count = int(count)
		}
		if accessKey, ok := item["access_key"].(string); ok {
			playlist.AccessKey = accessKey
		}
		if isFollowed, ok := item["is_followed"].(bool); ok {
			playlist.IsFollowed = isFollowed
		}

		if photoRaw, ok := item["photo"]; ok {
			if photoObj, ok := photoRaw.(map[string]interface{}); ok {
				if photoID, ok := photoObj["id"].(float64); ok {
					playlist.Photo.ID = int(photoID)
				}
				if url, ok := photoObj["photo_300"].(string); ok {
					playlist.Photo.Photo300 = url
				} else if url, ok := photoObj["photo_270"].(string); ok {
					playlist.Photo.Photo270 = url
				} else if url, ok := photoObj["photo_130"].(string); ok {
					playlist.Photo.Photo130 = url
				} else if url, ok := photoObj["photo_75"].(string); ok {
					playlist.Photo.Photo75 = url
				}
			}
		}

		playlists = append(playlists, playlist)
	}

	return playlists, nil
}

func getPlaylistTracks(token string, ownerID int, playlistID int, accessKey string) ([]Track, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("owner_id", strconv.Itoa(ownerID))
	params.Set("playlist_id", strconv.Itoa(playlistID))
	if accessKey != "" {
		params.Set("access_key", accessKey)
	}
	params.Set("count", "200")

	apiURL := getAPIBaseURL() + "audio.get?" + params.Encode()

	fmt.Printf("Requesting playlist tracks URL: %s\n", apiURL)

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
		Response struct {
			Count int     `json:"count"`
			Items []Track `json:"items"`
		} `json:"response"`
		Error *struct {
			ErrorCode int    `json:"error_code"`
			ErrorMsg  string `json:"error_msg"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга: %v", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("VK API ошибка: %s (код %d)", result.Error.ErrorMsg, result.Error.ErrorCode)
	}

	if result.Response.Items == nil {
		return []Track{}, nil
	}

	fmt.Printf("Found %d tracks in playlist\n", len(result.Response.Items))
	return result.Response.Items, nil
}

// ========== Функции для работы с сообщениями ==========

func getConversations(token string, count int, offset int) (*ConversationsResponse, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	params.Set("extended", "1")

	apiURL := getAPIBaseURL() + "messages.getConversations?" + params.Encode()

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

	var result ConversationsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга: %v", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("VK API ошибка: %s (код %d)", result.Error.ErrorMsg, result.Error.ErrorCode)
	}

	return &result, nil
}

func getMessages(token string, peerID int, count int, offset int) (*MessagesResponse, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	params.Set("peer_id", strconv.Itoa(peerID))
	params.Set("extended", "1")

	apiURL := getAPIBaseURL() + "messages.getHistory?" + params.Encode()

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

	var result MessagesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга: %v", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("VK API ошибка: %s (код %d)", result.Error.ErrorMsg, result.Error.ErrorCode)
	}

	return &result, nil
}

func sendMessage(token string, peerID int, message string) error {
	client := &http.Client{Timeout: 15 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("peer_id", strconv.Itoa(peerID))
	params.Set("message", message)
	params.Set("random_id", strconv.FormatInt(time.Now().UnixNano(), 10))

	apiURL := getAPIBaseURL() + "messages.send?" + params.Encode()

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

	var result struct {
		Response int `json:"response"`
		Error    *struct {
			ErrorCode int    `json:"error_code"`
			ErrorMsg  string `json:"error_msg"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if result.Error != nil {
		return fmt.Errorf("VK API ошибка: %s", result.Error.ErrorMsg)
	}

	return nil
}

// ========== Функции для работы с онлайн-статусом (невидимка) ==========

func setOffline(token string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")

	apiURL := getAPIBaseURL() + "account.setOffline?" + params.Encode()

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

	var result SetOnlineResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("ошибка парсинга: %v", err)
	}

	if result.Error != nil {
		return fmt.Errorf("VK API ошибка: %s", result.Error.ErrorMsg)
	}

	return nil
}

func setOnline(token string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")

	apiURL := getAPIBaseURL() + "account.setOnline?" + params.Encode()

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

	var result SetOnlineResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("ошибка парсинга: %v", err)
	}

	if result.Error != nil {
		return fmt.Errorf("VK API ошибка: %s", result.Error.ErrorMsg)
	}

	return nil
}

// ========== Обработчики ==========

func apiSetProxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	useProxy = req.Enabled
	fmt.Printf("Proxy mode: %v\n", useProxy)

	json.NewEncoder(w).Encode(map[string]string{"success": "true", "proxy_enabled": strconv.FormatBool(useProxy)})
}

func apiGetProxyStatusHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]bool{"proxy_enabled": useProxy})
}

func apiFriendTracksHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")
	userIDStr := r.URL.Query().Get("user_id")

	if userIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id required"})
		return
	}

	userID, _ := strconv.Atoi(userIDStr)

	tracks, err := getAllTracks(tokenCookie.Value, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

func apiGroupTracksHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")
	groupIDStr := r.URL.Query().Get("group_id")

	if groupIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "group_id required"})
		return
	}

	groupID, _ := strconv.Atoi(groupIDStr)
	ownerID := -groupID

	tracks, err := getAllTracks(tokenCookie.Value, ownerID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

func apiPlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, err := r.Cookie("vk_token")
	if err != nil || tokenCookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	userIDCookie, err := r.Cookie("vk_user_id")
	if err != nil || userIDCookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id not found"})
		return
	}

	userID, err := strconv.Atoi(userIDCookie.Value)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid user_id"})
		return
	}

	playlists, err := getPlaylists(tokenCookie.Value, userID)
	if err != nil {
		fmt.Printf("Error getting playlists: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if playlists == nil {
		playlists = []Playlist{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlists)
}

func apiPlaylistTracksHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")
	playlistIDStr := r.URL.Query().Get("playlist_id")
	ownerIDStr := r.URL.Query().Get("owner_id")
	accessKey := r.URL.Query().Get("access_key")

	fmt.Printf("=== Playlist Tracks Request ===\n")
	fmt.Printf("playlist_id: %s\n", playlistIDStr)
	fmt.Printf("owner_id: %s\n", ownerIDStr)
	fmt.Printf("access_key: %s\n", accessKey)

	if playlistIDStr == "" || ownerIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "playlist_id and owner_id required"})
		return
	}

	playlistID, _ := strconv.Atoi(playlistIDStr)
	ownerID, _ := strconv.Atoi(ownerIDStr)

	tracks, err := getPlaylistTracks(tokenCookie.Value, ownerID, playlistID, accessKey)
	if err != nil {
		fmt.Printf("Error getting playlist tracks: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	fmt.Printf("Returning %d tracks\n", len(tracks))

	if tracks == nil {
		tracks = []Track{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

func apiSetOfflineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tokenCookie, _ := r.Cookie("vk_token")
	if tokenCookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	err := setOffline(tokenCookie.Value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

func apiSetOnlineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tokenCookie, _ := r.Cookie("vk_token")
	if tokenCookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	err := setOnline(tokenCookie.Value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

func getFriends(token string, count int, offset int) (*FriendsResponse, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	params.Set("fields", "photo_100,online")

	apiURL := getAPIBaseURL() + "friends.get?" + params.Encode()

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

	var result FriendsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга: %v", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("VK API ошибка: %s (код %d)", result.Error.ErrorMsg, result.Error.ErrorCode)
	}

	return &result, nil
}

func getGroups(token string, count int, offset int) (*GroupsResponse, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	params := url.Values{}
	params.Set("access_token", token)
	params.Set("v", "5.131")
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	params.Set("extended", "1")
	params.Set("fields", "photo_100")

	apiURL := getAPIBaseURL() + "groups.get?" + params.Encode()

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

	var result GroupsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга: %v", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("VK API ошибка: %s (код %d)", result.Error.ErrorMsg, result.Error.ErrorCode)
	}

	return &result, nil
}

func vkCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	errorMsg := r.URL.Query().Get("error")
	if errorMsg != "" {
		fmt.Printf("VK Callback error: %s\n", errorMsg)
		http.Redirect(w, r, "/login?error="+errorMsg, http.StatusSeeOther)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	apiURL := fmt.Sprintf("%saccess_token?client_id=54533272&client_secret=tPyNb8rQNMXaTHRNp4NZ&code=%s&redirect_uri=%s/auth/vk-callback&grant_type=authorization_code",
		getOAuthBaseURL(), code, getBaseURL(r))

	resp, err := client.Get(apiURL)
	if err != nil {
		http.Redirect(w, r, "/login?error=network", http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Redirect(w, r, "/login?error=read", http.StatusSeeOther)
		return
	}

	var result struct {
		AccessToken string `json:"access_token"`
		UserId      int    `json:"user_id"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		http.Redirect(w, r, "/login?error=parse", http.StatusSeeOther)
		return
	}

	if result.Error != "" {
		http.Redirect(w, r, "/login?error="+result.Error, http.StatusSeeOther)
		return
	}

	if result.AccessToken == "" {
		http.Redirect(w, r, "/login?error=no_token", http.StatusSeeOther)
		return
	}

	userInfo, err := getUserInfo(result.AccessToken)
	if err != nil {
		http.Redirect(w, r, "/login?error=invalid_token", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_token",
		Value:    result.AccessToken,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_user_id",
		Value:    strconv.Itoa(result.UserId),
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_user_name",
		Value:    userInfo.FirstName + " " + userInfo.LastName,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
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

	apiURL := getAPIBaseURL() + "audio.getById?" + params.Encode()

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

func apiRecommendationsHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")

	client := &http.Client{Timeout: 15 * time.Second}
	params := url.Values{}
	params.Set("access_token", tokenCookie.Value)
	params.Set("v", "5.131")
	params.Set("count", "100")

	apiURL := getAPIBaseURL() + "audio.getRecommendations?" + params.Encode()

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

func apiProfileHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")

	client := &http.Client{Timeout: 10 * time.Second}
	params := url.Values{}
	params.Set("access_token", tokenCookie.Value)
	params.Set("v", "5.131")
	params.Set("fields", "photo_100")

	apiURL := getAPIBaseURL() + "users.get?" + params.Encode()

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

	var result struct {
		Response []struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Photo100  string `json:"photo_100"`
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
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("VK API ошибка: %s", result.Error.ErrorMsg)})
		return
	}

	if len(result.Response) == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Пользователь не найден"})
		return
	}

	user := result.Response[0]
	profile := map[string]interface{}{
		"id":         user.ID,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"photo":      user.Photo100,
		"full_name":  user.FirstName + " " + user.LastName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func apiSearchMyTracksHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")
	userIDCookie, _ := r.Cookie("vk_user_id")
	userID, _ := strconv.Atoi(userIDCookie.Value)

	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode([]Track{})
		return
	}

	allTracks, err := getAllTracks(tokenCookie.Value, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	queryLower := strings.ToLower(query)
	filtered := []Track{}
	for _, track := range allTracks {
		if strings.Contains(strings.ToLower(track.Title), queryLower) ||
			strings.Contains(strings.ToLower(track.Artist), queryLower) {
			filtered = append(filtered, track)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

func apiConversationsHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")

	if tokenCookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	conversations, err := getConversations(tokenCookie.Value, 200, 0)
	if err != nil {
		fmt.Printf("Error getting conversations: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	profilesMap := make(map[int]struct {
		FirstName string
		LastName  string
	})
	for _, profile := range conversations.Response.Profiles {
		profilesMap[profile.ID] = struct {
			FirstName string
			LastName  string
		}{profile.FirstName, profile.LastName}
	}

	result := make([]map[string]interface{}, 0)
	for _, item := range conversations.Response.Items {
		peerID := item.Conversation.Peer.ID

		title := ""
		if peerID > 0 {
			if profile, ok := profilesMap[peerID]; ok {
				title = profile.FirstName + " " + profile.LastName
			} else {
				title = fmt.Sprintf("Dialog %d", peerID)
			}
		} else {
			title = "Unknown dialog"
		}

		result = append(result, map[string]interface{}{
			"peer_id":   peerID,
			"title":     title,
			"unread":    item.Conversation.UnreadCount,
			"last_text": item.LastMessage.Text,
			"last_date": item.LastMessage.Date,
			"last_from": item.LastMessage.FromID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func apiMessagesHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")
	peerIDStr := r.URL.Query().Get("peer_id")

	if peerIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "peer_id required"})
		return
	}

	peerID, _ := strconv.Atoi(peerIDStr)

	messages, err := getMessages(tokenCookie.Value, peerID, 100, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	profilesMap := make(map[int]struct {
		FirstName string
		LastName  string
	})
	for _, profile := range messages.Response.Profiles {
		profilesMap[profile.ID] = struct {
			FirstName string
			LastName  string
		}{profile.FirstName, profile.LastName}
	}

	result := make([]map[string]interface{}, 0)
	for _, msg := range messages.Response.Items {
		senderName := ""
		if profile, ok := profilesMap[msg.FromID]; ok {
			senderName = profile.FirstName + " " + profile.LastName
		}
		if senderName == "" {
			senderName = fmt.Sprintf("User %d", msg.FromID)
		}

		result = append(result, map[string]interface{}{
			"id":        msg.ID,
			"date":      msg.Date,
			"from_id":   msg.FromID,
			"from_name": senderName,
			"text":      msg.Text,
			"out":       msg.Out,
			"peer_id":   msg.PeerID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func apiSendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tokenCookie, _ := r.Cookie("vk_token")

	var req struct {
		PeerID  int    `json:"peer_id"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	err := sendMessage(tokenCookie.Value, req.PeerID, req.Message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

func apiFriendsHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")

	if tokenCookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	friends, err := getFriends(tokenCookie.Value, 500, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	result := make([]map[string]interface{}, 0)
	for _, friend := range friends.Response.Items {
		result = append(result, map[string]interface{}{
			"id":         friend.ID,
			"first_name": friend.FirstName,
			"last_name":  friend.LastName,
			"full_name":  friend.FirstName + " " + friend.LastName,
			"photo":      friend.Photo100,
			"online":     friend.Online,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func apiGroupsHandler(w http.ResponseWriter, r *http.Request) {
	tokenCookie, _ := r.Cookie("vk_token")

	if tokenCookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	groups, err := getGroups(tokenCookie.Value, 500, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	result := make([]map[string]interface{}, 0)
	for _, group := range groups.Response.Items {
		result = append(result, map[string]interface{}{
			"id":          group.ID,
			"name":        group.Name,
			"screen_name": group.ScreenName,
			"photo":       group.Photo100,
			"type":        group.Type,
			"is_closed":   group.IsClosed,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "index.html")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "login.html")
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
				<h2>Ошибка авторизации</h2>
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
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_user_id",
		Value:    strconv.Itoa(userInfo.ID),
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400 * 30,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "vk_user_name",
		Value:    userInfo.FirstName + " " + userInfo.LastName,
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
	http.HandleFunc("/auth/vk-callback", vkCallbackHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/download", authMiddleware(downloadTrackHandler))
	http.HandleFunc("/get-track-url", authMiddleware(getTrackURLHandler))
	http.HandleFunc("/api/tracks", authMiddleware(apiTracksHandler))
	http.HandleFunc("/api/search", authMiddleware(apiSearchHandler))
	http.HandleFunc("/api/search-my", authMiddleware(apiSearchMyTracksHandler))
	http.HandleFunc("/api/add", authMiddleware(apiAddTrackHandler))
	http.HandleFunc("/api/delete", authMiddleware(apiDeleteTrackHandler))
	http.HandleFunc("/api/user", authMiddleware(apiUserInfoHandler))
	http.HandleFunc("/api/recommendations", authMiddleware(apiRecommendationsHandler))
	http.HandleFunc("/api/profile", authMiddleware(apiProfileHandler))
	http.HandleFunc("/api/conversations", authMiddleware(apiConversationsHandler))
	http.HandleFunc("/api/messages", authMiddleware(apiMessagesHandler))
	http.HandleFunc("/api/send-message", authMiddleware(apiSendMessageHandler))
	http.HandleFunc("/api/friends", authMiddleware(apiFriendsHandler))
	http.HandleFunc("/api/groups", authMiddleware(apiGroupsHandler))
	http.HandleFunc("/api/friend-tracks", authMiddleware(apiFriendTracksHandler))
	http.HandleFunc("/api/group-tracks", authMiddleware(apiGroupTracksHandler))
	http.HandleFunc("/api/playlists", authMiddleware(apiPlaylistsHandler))
	http.HandleFunc("/api/playlist-tracks", authMiddleware(apiPlaylistTracksHandler))
	http.HandleFunc("/api/set-offline", authMiddleware(apiSetOfflineHandler))
	http.HandleFunc("/api/set-online", authMiddleware(apiSetOnlineHandler))
	http.HandleFunc("/api/set-proxy", authMiddleware(apiSetProxyHandler))
	http.HandleFunc("/api/get-proxy-status", authMiddleware(apiGetProxyStatusHandler))

	port := "8080"
	fmt.Println("==================================================")
	fmt.Println("VK MOOSIC WEB PLAYER")
	fmt.Println("==================================================")
	fmt.Printf("\n✅ Сервер запущен: http://localhost:%s\n", port)
	fmt.Println("🌐 Открой в браузере: http://localhost:8080")
	fmt.Println("🔐 Доступен вход через ВКонтакте или по токену")
	fmt.Println("💬 Доступны диалоги и сообщения")
	fmt.Println("👥 Доступны друзья и сообщества")
	fmt.Println("📀 Доступны аудиозаписи")
	fmt.Println("📋 Доступны плейлисты")
	fmt.Println("🌐 Прокси для обхода блокировок (можно включить в настройках)")
	fmt.Println("📥 Треки можно скачать через кнопку меню")
	fmt.Println("⚠️ Удаление треков реально удаляет их из ВКонтакте!")
	fmt.Println("📌 Нажми Ctrl+C для остановки")
	fmt.Println("==================================================")

	http.ListenAndServe(":"+port, nil)
}
