package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"rss-reader/globals"
	"rss-reader/models"
	"rss-reader/utils"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func init() {
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetLevel(logrus.InfoLevel)
	globals.Init()
}

func main() {
	go utils.UpdateFeeds()
	go utils.WatchConfigFileChanges("config.json")
	http.HandleFunc("/feeds", getFeedsHandler)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/", tplHandler)

	// 加载静态文件
	fs := http.FileServer(http.FS(globals.DirStatic))
	http.Handle("/static/", fs)

	logger.Info("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func tplHandler(w http.ResponseWriter, r *http.Request) {
	tmplInstance := template.New("index.html").Delims("<<", ">>")
	funcMap := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}

	tmpl, err := tmplInstance.Funcs(funcMap).ParseFS(globals.DirStatic, "static/index.html")
	if err != nil {
		logger.Error("Template load error:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	formattedTime := time.Now().Format("15:04:05")
	darkMode := false
	nightStart := globals.RssUrls.NightStartTime
	nightEnd := globals.RssUrls.NightEndTime
	if nightStart != "" && nightEnd != "" {
		if nightStart < nightEnd {
			if formattedTime >= nightStart && formattedTime <= nightEnd {
				darkMode = true
			}
		} else {
			if formattedTime >= nightStart || formattedTime <= nightEnd {
				darkMode = true
			}
		}
	}

	data := struct {
		Keywords       string
		RssDataList    []models.Feed
		DarkMode       bool
		AutoUpdatePush int
	}{
		Keywords:       getKeywords(),
		RssDataList:    utils.GetFeeds(),
		DarkMode:       darkMode,
		AutoUpdatePush: globals.RssUrls.AutoUpdatePush,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		logger.Error("Template render error:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := globals.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	logger.Info("WebSocket connection established")
	updates := make(chan []byte)

	go func() {
		for _, url := range globals.RssUrls.Values {
			globals.Lock.RLock()
			cache, ok := globals.DbMap[url]
			globals.Lock.RUnlock()
			if !ok {
				logger.Warnf("Feed not found in db: %v", url)
				continue
			}
			data, err := json.Marshal(cache)
			if err != nil {
				logger.Error("JSON marshal error:", err)
				continue
			}
			updates <- data
		}
		close(updates)
	}()

	for data := range updates {
		err = conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			logger.Warn("WebSocket write error or connection closed:", err)
			return
		}
	}

	if globals.RssUrls.AutoUpdatePush > 0 {
		for {
			time.Sleep(time.Duration(globals.RssUrls.AutoUpdatePush) * time.Minute)
			for _, url := range globals.RssUrls.Values {
				globals.Lock.RLock()
				cache, ok := globals.DbMap[url]
				globals.Lock.RUnlock()
				if !ok {
					logger.Warnf("Feed not found in db: %v", url)
					continue
				}
				data, err := json.Marshal(cache)
				if err != nil {
					logger.Error("JSON marshal error:", err)
					continue
				}
				err = conn.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					logger.Warn("WebSocket write error or connection closed:", err)
					return
				}
			}
		}
	}
}

func getKeywords() string {
	words := ""
	for _, url := range globals.RssUrls.Values {
		globals.Lock.RLock()
		cache, ok := globals.DbMap[url]
		globals.Lock.RUnlock()
		if !ok {
			logger.Warnf("Feed not found in db: %v", url)
			continue
		}
		if cache.Title != "" {
			words += cache.Title + ","
		}
	}
	return words
}

func getFeedsHandler(w http.ResponseWriter, r *http.Request) {
	feeds := utils.GetFeeds()
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(feeds)
	if err != nil {
		logger.Error("JSON encode error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
