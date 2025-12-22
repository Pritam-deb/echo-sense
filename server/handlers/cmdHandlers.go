package handlers

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	recognisingalgorithm "github.com/Pritam-deb/echo-sense/internals/recognisingAlgorithm"
	"github.com/Pritam-deb/echo-sense/internals/spotify"
	wavservice "github.com/Pritam-deb/echo-sense/internals/wavService"
	"github.com/Pritam-deb/echo-sense/utils"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

const SONGS_DIR = "songs"

var logger = utils.GetLogger()

func Download(url string) {
	//Song download logic will be here

	err := utils.CreateDirIfNotExist(SONGS_DIR)
	if err != nil {

		ctx := context.Background()
		logger.ErrorContext(ctx, "Failed to create directory for songs", slog.Any("error", err))
	}

	if strings.Contains(url, "track") {
		spotify.DownloadSingleTrack(url, SONGS_DIR)
	}
}

func Search(track string) {
	filePath := filepath.Join(SONGS_DIR, track+".m4a")
	if _, err := os.Stat(filePath); err != nil {
		logger.Info("File doesnt exist.", "filepath", filePath)
		return
	}
	wavFilePath, err := wavservice.ConvertToWav(filePath, 1)
	if err != nil {
		logger.Error("Failed to convert to WAV", "error", err, "audioFilePath", filePath)
		return
	}
	wavInfo, err := wavservice.ReadWavFile(wavFilePath)
	if err != nil {
		logger.Error("Failed to read WAV file", "error", err, "wavFilePath", wavFilePath)
	}
	samples, err := wavservice.ConvertWavDataToSamples(wavInfo.Data)
	if err != nil {
		logger.Error("Failed to convert WAV data to samples", "error", err, "wavFilePath", wavFilePath)
		return
	}
	fmt.Println("Number of samples:", len(samples))
	matches, searchDuration, err := recognisingalgorithm.FindMatches(int(wavInfo.SampleRate), wavInfo.Duration, samples)

	if err != nil {
		fmt.Println("Error finding matches:", err)
		return
	}

	if len(matches) == 0 {
		fmt.Println("\nNo match found.")
		fmt.Printf("\nSearch took: %s\n", searchDuration)
		return
	}

	msg := "Matches:"
	topMatches := matches
	if len(matches) >= 5 {
		msg = "Top 5 matches:"
		topMatches = matches[:5]
	}

	fmt.Println(msg)
	for _, match := range topMatches {
		fmt.Printf("\t- %s by %s, score: %.2f\n",
			match.SongTitle, match.SongArtist, match.Score)
	}

	fmt.Printf("\nSearch took: %s\n", searchDuration)
	topMatch := topMatches[0]
	fmt.Printf("\nFinal prediction: %s by %s , score: %.2f\n", topMatch.SongTitle, topMatch.SongArtist, topMatch.Score)
}

func Serve(protocol string, port string) {
	protocol = strings.ToLower(protocol)
	var allowOriginFunc = func(r *http.Request) bool {
		return true
	}

	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: allowOriginFunc,
			},
			&websocket.Transport{
				CheckOrigin: allowOriginFunc,
			},
		},
	})

	server.OnConnect("/", func(socket socketio.Conn) error {
		socket.SetContext("")
		log.Println("CONNECTED: ", socket.ID())

		return nil
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("closed", reason)
	})

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	serveHTTPS := protocol == "https"
	serveHTTP(server, serveHTTPS, port)

}

func serveHTTP(socketServer *socketio.Server, serveHTTPS bool, port string) {
	http.Handle("/socket.io/", socketServer)
	http.Handle("/", http.FileServer(http.Dir("static")))

	if serveHTTPS {
		httpsAddr := ":" + port
		httpsServer := &http.Server{
			Addr: httpsAddr,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			Handler: socketServer,
		}

		cert_key_default := "/etc/letsencrypt/live/localport.online/privkey.pem"
		cert_file_default := "/etc/letsencrypt/live/localport.online/fullchain.pem"

		cert_key := utils.GetEnv("CERT_KEY", cert_key_default)
		cert_file := utils.GetEnv("CERT_FILE", cert_file_default)
		if cert_key == "" || cert_file == "" {
			log.Fatal("Missing cert")
		}

		log.Printf("Starting HTTPS server on %s\n", httpsAddr)
		if err := httpsServer.ListenAndServeTLS(cert_file, cert_key); err != nil {
			log.Fatalf("HTTPS server ListenAndServeTLS: %v", err)
		}
	}

	log.Printf("Starting HTTP server on port %v", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}
