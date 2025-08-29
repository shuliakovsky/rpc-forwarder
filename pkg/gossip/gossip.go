package gossip

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/shuliakovsky/rpc-forwarder/pkg/peers"
	"go.uber.org/zap"
)

type GossipMessage struct {
	From  string       `json:"from"`
	Peers []peers.Peer `json:"peers"`
}

func Start(store *peers.Store, selfID string, logger *zap.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		plist := store.List()
		if len(plist) <= 1 {
			continue
		}

		// Выбираем случайного пира, кроме себя
		target := plist[rand.Intn(len(plist))]
		if target.ID == selfID {
			continue
		}

		msg := GossipMessage{From: selfID, Peers: plist}
		data, _ := json.Marshal(msg)

		url := "http://" + target.Addr + "/gossip"
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			logger.Warn("Ошибка gossip отправки", zap.String("target", target.ID), zap.Error(err))
			continue
		}
		resp.Body.Close()

		logger.Debug("Gossip отправлен", zap.String("to", target.ID), zap.Int("peers_count", len(plist)))
	}
}

// Обработчик входящих gossip-сообщений
func Handler(store *peers.Store, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var msg GossipMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Мержим список
		for _, p := range msg.Peers {
			store.Add(p)
		}

		logger.Debug("Gossip получен", zap.String("from", msg.From), zap.Int("count", len(msg.Peers)))
		w.WriteHeader(http.StatusOK)
	}
}
