package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
)

var (
	err           error
	audioFileName = "output.ogg"
	oggPageDuration = time.Millisecond * 20
)

func main() {
	if len(os.Args) != 2 {
		panic("IVSStageSaver requires a Token")
	}
	bearerToken := os.Args[1]

	fmt.Println("have bearer token")

	// Assert that we have an audio file
	_, err = os.Stat(audioFileName)
	haveAudioFile := !os.IsNotExist(err)

	if !haveAudioFile {
		panic("Could not find `" + audioFileName + "`")
	}
	fmt.Println("have file")

	_, err = createPeerConnection("https://global.whip.live-video.net", bearerToken, func(peerConnection *webrtc.PeerConnection) error {

		iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

		if haveAudioFile {
			// Create a audio track
			audioTrack, audioTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
			if audioTrackErr != nil {
				panic(audioTrackErr)
			}

			rtpSender, audioTrackErr := peerConnection.AddTrack(audioTrack)
			if audioTrackErr != nil {
				panic(audioTrackErr)
			}

			// Read incoming RTCP packets
			// Before these packets are returned they are processed by interceptors. For things
			// like NACK this needs to be called.
			go func() {
				rtcpBuf := make([]byte, 1500)
				for {
					if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
						return
					}
				}
			}()

			go func() {
				// Open a ogg file and start reading using our oggReader
				file, oggErr := os.Open(audioFileName)
				if oggErr != nil {
					panic(oggErr)
				}

				// Open on oggfile in non-checksum mode.
				ogg, _, oggErr := oggreader.NewWith(file)
				if oggErr != nil {
					panic(oggErr)
				}

				// Wait for connection established
				<-iceConnectedCtx.Done()

				// Keep track of last granule, the difference is the amount of samples in the buffer
				var lastGranule uint64

				// It is important to use a time.Ticker instead of time.Sleep because
				// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
				// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
				ticker := time.NewTicker(oggPageDuration)
				for ; true; <-ticker.C {
					pageData, pageHeader, oggErr := ogg.ParseNextPage()
					if errors.Is(oggErr, io.EOF) {
						fmt.Printf("All audio pages parsed and sent")
						os.Exit(0)
					}

					if oggErr != nil {
						panic(oggErr)
					}

					// The amount of samples is the difference between the last and current timestamp
					sampleCount := float64(pageHeader.GranulePosition - lastGranule)
					lastGranule = pageHeader.GranulePosition
					sampleDuration := time.Duration((sampleCount/48000)*1000) * time.Millisecond

					if oggErr = audioTrack.WriteSample(media.Sample{Data: pageData, Duration: sampleDuration}); oggErr != nil {
						panic(oggErr)
					}
				}
			}()
		}

		// It is necessary to create and add a video track in order to establish a connection
		videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
		if videoTrackErr != nil {
			panic(videoTrackErr)
		}
		_, videoTrackErr = peerConnection.AddTrack(videoTrack)
		if videoTrackErr != nil {
			panic(videoTrackErr)
		}

		// Set the handler for ICE connection state
		// This will notify you when the peer has connected/disconnected
		peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			fmt.Printf("Connection State has changed %s \n", connectionState.String())
			if connectionState == webrtc.ICEConnectionStateConnected {
				iceConnectedCtxCancel()
			}
		})

		// Set the handler for Peer connection state
		// This will notify you when the peer has connected/disconnected
		peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
			fmt.Printf("Peer Connection State has changed: %s\n", connectionState.String())

			if connectionState == webrtc.PeerConnectionStateFailed {
				// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
				// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
				// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
				fmt.Println("Peer Connection has gone to failed exiting")
				os.Exit(0)
			}
		})

		return nil
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to createPeerConnection %s", err.Error()))
	}

	select {}
}
