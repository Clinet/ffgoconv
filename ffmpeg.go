package ffgoconv

import (
	"fmt"
	"errors"
	"io"
	"io/ioutil"
	"os/exec"
)

var (
	ErrFFmpegNotRunning    = errors.New("ffgoconv: FFmpeg: not running")
	ErrFFmpegFilepathEmpty = errors.New("ffgoconv: filepath must not be empty")
)

// FFmpeg contains all the data required to keep an FFmpeg process running and usable.
type FFmpeg struct {
	process *exec.Cmd
	closed  bool
	err     error
	
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// NewFFmpeg returns an initialized *FFmpeg or an error if one could not be created.
//
// If filepath is empty, the FFmpeg process will not start. You can specify any location supported by FFmpeg, such as a network location or a local filepath.
//
// If args is nil or empty, the default values will be used. Do not specify your own arguments unless you understand how ffgoconv functions.
func NewFFmpeg(filepath string, args []string) (*FFmpeg, error) {
	if filepath == "" {
		return nil, ErrFFmpegFilepathEmpty
	}
	if len(args) == 0 {
		args = []string{
			"-hide_banner",
			"-stats",
			"-re", "-i", filepath,
			"-map", "0:a",
			"-acodec", "pcm_f64le",
			"-f", "f64le",
			"-vol", "256",
			"-ar", "48000",
			"-ac", "2",
			"-threads", "1",
			"pipe:1",
		}
	}
	
	ffmpeg := exec.Command("ffmpeg", args...)
	
	stdinPipe, err := ffmpeg.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := ffmpeg.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrPipe, err := ffmpeg.StderrPipe()
	if err != nil {
		return nil, err
	}
	
	err = ffmpeg.Start()
	if err != nil {
		stdoutData, _ := ioutil.ReadAll(stdoutPipe)
		stderrData, _ := ioutil.ReadAll(stderrPipe)
		
		stdinPipe.Close()
		stdoutPipe.Close()
		stderrPipe.Close()
		
		err = fmt.Errorf("ffgoconv: FFmpeg: error starting process: {err: \"%v\", stdout: \"%v\", stderr: \"%v\"}", err, stdoutData, stderrData)
		return nil, err
	}

	ff := &FFmpeg{
		process: ffmpeg,
		stdin: stdinPipe,
		stdout: stdoutPipe,
		stderr: stderrPipe,
	}

	go func() {
		if err := ffmpeg.Wait(); err != nil {
			stdoutData, _ := ioutil.ReadAll(ff.stdout)
			stderrData, _ := ioutil.ReadAll(ff.stderr)

			ff.err = fmt.Errorf("ffgoconv: FFmpeg: error starting process: {err: \"%v\", stdout: \"%v\", stderr: \"%v\", args: \"%v\"}", err, stdoutData, stderrData, args)
		}
		ff.Close()
	}()

	return ff, nil
}

// IsRunning returns whether or not the FFmpeg process is running, per the knowledge of ffgoconv.
func (ff *FFmpeg) IsRunning() bool {
	return !ff.closed
}

// Close closes the FFmpeg process gracefully and renders the struct unusable.
func (ff *FFmpeg) Close() {
	if ff.closed {
		return
	}
	ff.process.Process.Kill()
	ff.stdin.Close()
	ff.stdout.Close()
	ff.stderr.Close()
	ff.closed = true
}

// Err returns the last stored error. Error histories are not kept, so check as soon as something goes wrong.
func (ff *FFmpeg) Err() error {
	return ff.err
}

func (ff *FFmpeg) setError(err error) {
	ff.err = err
}

// Read implements an io.Reader wrapper around *FFmpeg.stdout.
func (ff *FFmpeg) Read(data []byte) (n int, err error) {
	if !ff.IsRunning() {
		return 0, ErrFFmpegNotRunning
	}
	
	n, err = ff.stdout.Read(data)
	if err != nil {
		ff.Close()
	}
	return n, err
}

// ReadError implements an io.Reader wrapper around *FFmpeg.stderr.
func (ff *FFmpeg) ReadError(data []byte) (n int, err error) {
	if !ff.IsRunning() {
		return 0, ErrFFmpegNotRunning
	}
	
	n, err = ff.stderr.Read(data)
	if err != nil {
		ff.Close()
	}
	return n, err
}

// Write implements an io.Writer wrapper around *FFmpeg.stdin.
func (ff *FFmpeg) Write(data []byte) error {
	if !ff.IsRunning() {
		return ErrFFmpegNotRunning
	}
	
	_, err := ff.stdin.Write(data)
	return err
}

