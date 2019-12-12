package ffgoconv

import (
	"encoding/binary"
	"errors"
	//"fmt"
	"math"
)

// Streamer contains all the data required to run a streaming session.
type Streamer struct {
	FFmpeg *FFmpeg
	Volume float64
	
	err error
}

func NewStreamer(filepath string, args []string, volume float64) (*Streamer, error) {
	ffmpeg, err := NewFFmpeg(filepath, args)
	if err != nil {
		return nil, err
	}
	
	return &Streamer{
		FFmpeg: ffmpeg,
		Volume: volume,
	}, nil
}

func (streamer *Streamer) IsRunning() bool {
	return streamer.FFmpeg.IsRunning()
}

func (streamer *Streamer) Close() {
	if streamer != nil {
		if streamer.FFmpeg != nil {
			streamer.FFmpeg.Close()
		}
	}
}

// Err returns the last stored error. Error histories are not kept, so check as soon as something goes wrong.
func (streamer *Streamer) Err() error {
	return streamer.err
}

func (streamer *Streamer) setError(err error) {
	streamer.err = err
}

// ReadSample returns the next audio sample from the streaming session.
func (streamer *Streamer) ReadSample() (float64, error) {
	if !streamer.IsRunning() {
		return 0, errors.New("ffgoconv: streamer: not running")
	}
	
	sample := make([]byte, 8) // sizeof(float64) == 8
	
	n, err := streamer.FFmpeg.Read(sample)
	if err != nil {
		return 0, err
	}
	if n != 8 {
		return 0, errors.New("streamer: read: size of sample must be 8")
	}
	
	u64 := binary.LittleEndian.Uint64(sample)
	fSample := math.Float64frombits(u64)
	
	//fmt.Println("sample:f64 (", fSample, fSample * streamer.Volume, ")")
	
	fSample *= streamer.Volume
	
	/*errFF := make([]byte, 0)
	for {
		tmp := make([]byte, 1)
		n, _ := streamer.FFmpeg.ReadError(tmp)
		if n == 0 {
			break
		}
		errFF = append(errFF, tmp[0])
	}
	if len(errFF) > 0 {
		fmt.Println(string(errFF))
	}*/
	
	return fSample, nil
}

// WriteSample writes a new audio sample to the streaming session.
func (streamer *Streamer) WriteSample(sample float64) error {
	if !streamer.IsRunning() {
		return errors.New("ffgoconv: streamer: not running")
	}
	
	var bs [8]byte
	u64 := math.Float64bits(sample)
	binary.LittleEndian.PutUint64(bs[:], u64)
	
	err := streamer.FFmpeg.Write(bs[:])
	return err
}

func (streamer *Streamer) SetVolume(volume float64) {
	streamer.Volume = volume
}
