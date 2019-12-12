package ffgoconv

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

// Transmuxer contains all the data required to run a transmuxing session.
type Transmuxer struct {
	sync.Mutex
	
	Streamers []*Streamer
	FinalStream *Streamer
	MasterVolume float64
	
	running bool
	err     error
}

// NewTransmuxer returns an initialized *Transmuxer or an error if one could not be created.
//
// If streamers is nil, it will be initialized automatically with an empty slice of *Streamer.
//
// If codec is not specified, the ffmpeg process will not start. A list of possible codecs can be found with "ffmpeg -codecs".
//
// If format is not specified, the ffmpeg process will not start. A list of possible formats can be found with "ffmpeg -formats".
//
// If bitrate is not specified, the ffmpeg process will not start.
//
// The variable masterVolume must be a floating-point number between 0 and 1, representing a percentage value. For example, 20% volume would be 0.2.
//
// If outputFilepath is empty, a buffer of float64 PCM values will be initialized and the returned *Transmuxer can be then used as an io.Reader.
//
// If outputFilepath is "pipe:1", the FinalStream *Streamer can be used as an io.Reader to receive encoded audio data of the chosen codec in the chosen format.
func NewTransmuxer(streamers []*Streamer, outputFilepath, codec, format, bitrate string, masterVolume float64) (*Transmuxer, error) {
	if streamers == nil {
		streamers = make([]*Streamer, 0)
	}

	args := []string{
		"-hide_banner",
		"-stats",
		"-acodec", "pcm_f64le",
		"-f", "f64le",
		"-ar", "48000",
		"-ac", "2",
		"-re", "-i", "-",
		"-acodec", codec,
		"-f", format,
		"-vol", "256",
		"-ar", "48000",
		"-ac", "2",
		"-b:a", bitrate,
		"-threads", "2",
		outputFilepath,
	}

	var finalStream *Streamer
	var err error

	if outputFilepath != "" {
		finalStream, err = NewStreamer(outputFilepath, args, 1.0)
		if err != nil {
			return nil, err
		}

		return &Transmuxer{
			Streamers:    streamers,
			FinalStream:  finalStream,
			MasterVolume: masterVolume,
		}, nil
	}

	return &Transmuxer{
		Streamers:    streamers,
		MasterVolume: masterVolume,
		//buffer:       make([]float64, 0),
	}, nil
}

// Read implements an io.Reader wrapper around *Transmuxer.FinalStream.FFmpeg.stdout.
func (transmuxer *Transmuxer) Read(data []byte) (n int, err error) {
	
	

	if !transmuxer.IsRunning() {
		return 0, io.EOF
	}
	
	n, err = transmuxer.FinalStream.FFmpeg.Read(data)
	
	return n, err
}

// AddStreamer initializes and adds a *Streamer to the transmuxing session, or returns an error if one could not be initialized.
// See NewStreamer for info on supported arguments.
func (transmuxer *Transmuxer) AddStreamer(filepath string, args []string, volume float64) (*Streamer, error) {
	
	

	streamer, err := NewStreamer(filepath, args, volume)
	if err != nil {
		return nil, err
	}

	transmuxer.Streamers = append(transmuxer.Streamers, streamer)

	return streamer, nil
}

// AddRunningStreamer adds a pre-initialized *Streamer to the transmuxing session
func (transmuxer *Transmuxer) AddRunningStreamer(streamer *Streamer) (*Streamer, error) {
	
	

	if !transmuxer.IsRunning() {
		return nil, errors.New("ffgoconv: transmuxer: not running")
	}

	transmuxer.Streamers = append(transmuxer.Streamers, streamer)
	return streamer, nil
}

func (transmuxer *Transmuxer) SetMasterVolume(volume float64) {
	
	

	transmuxer.MasterVolume = volume
}

func (transmuxer *Transmuxer) IsRunning() bool {
	
	
	return transmuxer.running
}

func (transmuxer *Transmuxer) Close() {
	
	

	if !transmuxer.IsRunning() {
		return
	}
	
	for _, streamer := range transmuxer.Streamers {
		streamer.Close()
	}
}

// Err returns the last stored error. Error histories are not kept, so check as soon as something goes wrong.
func (transmuxer *Transmuxer) Err() error {
	
	

	return transmuxer.err
}

func (transmuxer *Transmuxer) setError(err error) {
	
	

	transmuxer.err = err
}

func (transmuxer *Transmuxer) Run() {
	if transmuxer.IsRunning() {
		return
	}
	
	//fmt.Println("NOW RUNNING")

	transmuxer.running = true

	for {
		
		

		var sample float64
		
		continuedStreamers := make([]*Streamer, 0)
		for _, streamer := range transmuxer.Streamers {
			newSample, err := streamer.ReadSample()
			if err != nil {
				fmt.Println("CLOSING STREAMER:", err)
				streamer.setError(err)
				streamer.Close()
				continue
			}
			
			continuedStreamers = append(continuedStreamers, streamer)
			
			sample += newSample
		}
		
		if len(continuedStreamers) == 0 {
			break
		}
		
		transmuxer.Streamers = continuedStreamers
		
		sample = sample * transmuxer.MasterVolume
		
		if transmuxer.FinalStream != nil {
			err := transmuxer.FinalStream.WriteSample(sample)
			if err != nil {
				transmuxer.setError(err)
				transmuxer.Close()
				return
			}
		}
	}
	
	transmuxer.running = false
	transmuxer.Close()
}












































