package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

func createPeerConnection(url, bearerToken string, configureCallback func(peerConnection *webrtc.PeerConnection) error) (*webrtc.PeerConnection, error) {
	var (
		iceServers         []webrtc.ICEServer
		iceTransportPolicy = webrtc.ICETransportPolicyAll
		err                error
	)

	// Create a MediaEngine object to configure the supported codec
	m := &webrtc.MediaEngine{}

	// Create audio codec
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2, SDPFmtpLine: "minptime=10;useinbandfec=1", RTCPFeedback: nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, err
	}

	// Create video codec
	// videoRTCPFeedback := []webrtc.RTCPFeedback{{Type: "goog-remb", Parameter: ""}, {Type: "ccm", Parameter: "fir"}, {Type: "nack", Parameter: ""}, {Type: "nack", Parameter: "pli"}}
	// if err := m.RegisterCodec(webrtc.RTPCodecParameters{
	// 	RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: videoRTCPFeedback},
	// 	PayloadType:        96,
	// }, webrtc.RTPCodecTypeVideo); err != nil {
	// 	return nil, err
	// }

	// Appears to be just a way to generate reports
	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}

	// allows the user to override the default SRTP Protection Profiles
	s := webrtc.SettingEngine{}
	s.SetSRTPProtectionProfiles(dtls.SRTP_AES128_CM_HMAC_SHA1_80)
	s.SetRelayAcceptanceMinWait(time.Second)
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i), webrtc.WithSettingEngine(s))

	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers:         iceServers,
		ICETransportPolicy: iceTransportPolicy,
	})
	if err != nil {
		return nil, err
	}

	if err = configureCallback(peerConnection); err != nil {
		return nil, err
	}
	readyToOffer, readyToOfferCancel := context.WithCancel(context.Background())

	readyToOfferCancel()

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return nil, err
	}
	log.Println("offer created")

	if err := peerConnection.SetLocalDescription(offer); err != nil {
		return nil, err
	}

	<-readyToOffer.Done()
	if err := postOffer(bearerToken, url, peerConnection); err != nil {
		return nil, err
	}

	return peerConnection, nil
}

func postOffer(bearerToken, mediaServerURL string, peerConnection *webrtc.PeerConnection) error {
	req, err := http.NewRequest("POST", mediaServerURL, bytes.NewBuffer([]byte(peerConnection.LocalDescription().SDP)))
	if err != nil {
		return err
	}

	addToken(req, bearerToken)
	req.Header.Add("Content-Type", "application/sdp")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			addToken(req, bearerToken)
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("err", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("err", err)
	}
	log.Println("response body", string(body))

	return peerConnection.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: string(body)})
}

func addToken(req *http.Request, bearerToken string) {
	req.Header.Add("Authorization", "Bearer "+bearerToken)
}

func sendSilentAudio(audioTrack *webrtc.TrackLocalStaticSample) {
	audioDuration := 20 * time.Millisecond
	for ; true; <-time.NewTicker(audioDuration).C {
		// This is an example of how to write an audio track
		_ = audioTrack.WriteSample(media.Sample{Data: []byte{0xFF, 0xFF, 0xFE}, Duration: audioDuration})
	}
}
