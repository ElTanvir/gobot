package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
)

func main() {
	token := "MTEwOTQ1MzY5NjM5NzQ4ODE5OA.GxhGP0.8WQ-d5m9u6tm3fd3ttYUBjhW-KVRnKgpenkq8c"
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}
	dg.AddHandler(messageCreate)
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening Discord connection:", err)
	}
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	<-make(chan struct{})
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if strings.Contains(m.Content, "youtube.com") || strings.Contains(m.Content, "youtu.be") {
		log.Println("Processing YouTube link:", m.Content)

		videoID := extractVideoID(m.Content)
		log.Println("Extracted video ID:", videoID)

		client := &youtube.Client{}
		video, err := client.GetVideo(videoID)
		if err != nil {
			log.Println("Error getting video details:", err)
			return
		}
		log.Println("Retrieved video details:")

		var audioURL string

		for _, format := range video.Formats {
			log.Println("Format:", format.MimeType, "Quality:", format.AudioQuality, "Audio FrameRate:", format.AudioSampleRate, "Audio Bitrate:", format.Quality)
			fmt.Println()
			fmt.Println()
			if strings.Contains(format.MimeType, "audio/webm") && format.AudioQuality == "AUDIO_QUALITY_MEDIUM" {
				audioURL = format.URL
				break
			}
		}

		if audioURL == "" {
			log.Println("Error: Audio URL not found")
			return
		}

		options := dca.StdEncodeOptions
		options.RawOutput = true
		options.Bitrate = 128

		stream, err := dca.EncodeFile(audioURL, options)
		if err != nil {
			log.Println("Error creating audio stream reader:", err)
			return
		}
		defer stream.Cleanup()

		voiceChannelID := getVoiceChannelID(s, m.GuildID, m.Author.ID)
		if voiceChannelID == "" {
			log.Println("User not in a voice channel")
			return
		}
		log.Println("User voice channel ID:", voiceChannelID)

		voiceConnection, err := s.ChannelVoiceJoin(m.GuildID, voiceChannelID, false, true)
		if err != nil {
			log.Println("Error joining voice channel:", err)
			return
		}
		log.Println("Joined voice channel")

		voiceConnection.Speaking(true)
		log.Println("Started speaking in voice channel")

		for {
			frame, err := stream.OpusFrame()
			if err != nil {
				if err == io.EOF {
					// End of audio stream, break the loop
					log.Println("End of audio stream reached")
					break
				}
				log.Println("Error retrieving opus frame:", err)
				break
			}

			voiceConnection.OpusSend <- frame
		}

		voiceConnection.Speaking(false)
		log.Println("Stopped speaking in voice channel")
	} else {
		log.Println("Non-YouTube content:", m.Content)
	}
}

func extractVideoID(url string) string {
	parts := strings.Split(url, "=")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

func getVoiceChannelID(s *discordgo.Session, guildID string, userID string) string {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		return ""
	}
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			return vs.ChannelID
		}
	}
	return ""
}
