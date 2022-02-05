package airplay2

import (
	"ledfx/audio"
	"ledfx/color"
	"ledfx/handlers/player"
	"ledfx/handlers/raop"
	"ledfx/handlers/rtsp"
	"ledfx/integrations/airplay2/codec"
	log "ledfx/logger"
	"sync"
	"unsafe"
)

type audioPlayer struct {
	/* Variables that are looped through often belong at the top of the struct */
	wg sync.WaitGroup

	intWriter  audio.IntWriter
	byteWriter *audio.ByteWriter

	hasClients, hasDecodedOutputs, sessionActive, muted, doBroadcast bool

	numClients int
	apClients  [8]*Client

	quit chan bool

	artwork []byte
	album   string
	artist  string
	title   string

	volume float64
}

func newPlayer(intWriter audio.IntWriter, byteWriter *audio.ByteWriter) *audioPlayer {
	p := &audioPlayer{
		apClients:  [8]*Client{},
		volume:     1,
		quit:       make(chan bool),
		wg:         sync.WaitGroup{},
		intWriter:  intWriter,
		byteWriter: byteWriter,
	}

	return p
}

func (p *audioPlayer) Play(session *rtsp.Session) {
	log.Logger.WithField("category", "AirPlay Player").Warnf("Starting new session")
	p.sessionActive = true
	decoder := codec.GetCodec(session)
	go func(dc *codec.Handler) {
		defer func() {
			p.sessionActive = false
		}()
		for {
			select {
			case recvBuf, ok := <-session.DataChan:
				if !ok {
					return
				}
				if p.muted {
					continue
				}
				func() {
					defer func() {
						if err := recover(); err != nil {
							log.Logger.WithField("category", "AirPlay Player").Warnf("Recovered from panic during playStream: %v\n", err)
						}
					}()

					recvBuf = dc.Decode(recvBuf)
					codec.NormalizeAudio(recvBuf, p.volume)

					p.byteWriter.Write(recvBuf)

					p.intWriter.Write(bytesToAudioBufferUnsafe(recvBuf))
				}()
			case <-p.quit:
				log.Logger.WithField("category", "AirPlay Player").Warnf("Session with peer '%s' closed", session.Description.ConnectData.ConnectionAddress)
				return
			}
		}
	}(decoder)
}

func bytesToAudioBufferUnsafe(p []byte) (out audio.Buffer) {
	out = make([]int16, len(p))
	var offset int
	for i := 0; i < len(p); i += 2 {
		out[offset] = twoBytesToInt16Unsafe(p[i : i+2])
		offset++
	}
	return
}

func bytesToAudioBuffer(p []byte) (out audio.Buffer) {
	out = make([]int16, len(p))
	var offset int
	for i := 0; i < len(p); i += 2 {
		out[offset] = twoBytesToInt16(p[i : i+2])
		offset++
	}
	return
}

func twoBytesToInt16(p []byte) (out int16) {
	out |= int16(p[0])
	out |= int16(p[1]) << 8
	return
}

func twoBytesToInt16Unsafe(p []byte) (out int16) {
	return *(*int16)(unsafe.Pointer(&p[0]))
}

func (p *audioPlayer) AddClient(client *Client) {
	p.hasClients = true
	p.apClients[p.numClients] = client
	p.byteWriter.AppendWriter(p.apClients[p.numClients])
	p.numClients++
}

func (p *audioPlayer) SetVolume(volume float64) {
	p.volume = volume
	if p.hasClients {
		p.broadcastParam(raop.ParamVolume(prepareVolume(volume)))
	}
}

func (p *audioPlayer) SetMute(isMuted bool) {
	p.muted = isMuted
	if p.hasClients {
		p.broadcastParam(raop.ParamMuted(isMuted))
	}
	if isMuted {
		log.Logger.WithField("category", "AirPlay Player").Infoln("Muting stream...")
	}
}

func (p *audioPlayer) GetIsMuted() bool {
	return p.muted
}

func (p *audioPlayer) SetTrack(album string, artist string, title string) {
	p.album = album
	p.artist = artist
	p.title = title
	if p.hasClients {
		p.broadcastParam(raop.ParamTrackInfo{
			Album:  album,
			Artist: artist,
			Title:  title,
		})
	}
}

func (p *audioPlayer) SetAlbumArt(artwork []byte) {
	p.artwork = artwork
	if p.hasClients {
		p.broadcastParam(raop.ParamAlbumArt(artwork))
	}
}

func (p *audioPlayer) GetGradientFromArtwork(resolution int) (*color.Gradient, error) {
	return color.GradientFromPNG(p.artwork, resolution, 75)
}

func (p *audioPlayer) GetTrack() player.Track {
	return player.Track{
		Artist:  p.artist,
		Album:   p.album,
		Title:   p.title,
		Artwork: p.artwork,
	}
}

// airplay server will apply a normalization,
// we have the raw volume on a scale of 0 to 1,
// so we build the proper format. (-144 through 0)
func prepareVolume(vol float64) float64 {
	switch {
	case vol == 0:
		return -144
	case vol == 1:
		return 0
	default:
		return (vol * 30) - 30
	}
}

func (p *audioPlayer) Close() {
	if p.sessionActive {
		p.quit <- true
	}
}

func (p *audioPlayer) broadcastParam(par interface{}) {
	for i := 0; i < p.numClients; i++ {
		p.apClients[i].SetParam(par)
	}
}

func (p *audioPlayer) broadcastEncoded(data []byte) {
	for i := 0; i < p.numClients; i++ {
		_, _ = p.apClients[i].DataConn.Write(data)
	}
}
