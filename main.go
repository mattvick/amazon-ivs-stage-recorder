package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pion/webrtc/v3"
)

var (
	err error
)

func main() {
	if len(os.Args) != 2 {
		panic("IVSStageSaver requires a Token")
	}
	bearerToken := os.Args[1]

	log.SetPrefix("whip example: ")
	log.SetFlags(0)

	_, err = createPeerConnection("https://global.whip.live-video.net", bearerToken, func(peerConnection *webrtc.PeerConnection) error {

		audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
		if err != nil {
			panic(fmt.Sprintf("Failed to NewTrackLocalStaticSample %s", err.Error()))
		}
		go sendSilentAudio(audioTrack)

		if _, err = peerConnection.AddTrack(audioTrack); err != nil {
			panic(fmt.Sprintf("Failed to AddTrack %s", err.Error()))
		}
		log.Println("audio track added")

		// Create a video transceiver
		// if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}); err != nil {
		// 	panic(fmt.Sprintf("Failed to AddTransceiverFromKind %s", err.Error()))
		// }
		// log.Println("video transceiver added")

		peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
			fmt.Printf("Connection State has changed %s \n", connectionState.String())
		})

		return nil
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to createPeerConnection %s", err.Error()))
	}

	select {}
}
