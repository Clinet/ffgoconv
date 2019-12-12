package main

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Clinet/ffgoconv"
	"github.com/matryer/runner"
)

func main() {
	//Make sure sleep duration is specified
	if len(os.Args) < 2 {
		panic("You must specify a sleep duration in milliseconds!")
	}
	//Make sure audio files are specified in program args
	if len(os.Args) < 3 {
		panic("You must specify one or more audio files to transmux")
	}

	//Get sleep duration
	sleepDur, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}

	//Get list of specified files
	files := os.Args[2:]

	//Remove test file if exists
	os.Remove("./test.mp3")

	//Create slice of streamers
	streamers := make([]*ffgoconv.Streamer, 0)

	//Create new transmuxing session, tell it to output encoded data to final stream's stdout, use MP3 params at 320Kbps
	transmuxer, err := ffgoconv.NewTransmuxer(streamers, "pipe:1", "libmp3lame", "mp3", "128k", 1)
	if err != nil {
		panic(err)
		return
	}

	//Add all files to transmuxer
	/*if len(files) > 0 {
		for i, file := range files {
			log.Println("Adding stream [:", i+1, "]:", file)
			_, err = transmuxer.AddStreamer(file, nil, 1)
			if err != nil {
				panic(err)
			}
		}
	}*/

	music, err := transmuxer.AddStreamer(files[0], nil, 1)
	if err != nil {
		panic(err)
	}

	log.Println("Starting transmuxer...")
	go transmuxer.Run()

	log.Println("Creating reading task...")
	test := make([]byte, 0)

	task := runner.Go(func(shouldStop runner.S) error {
		for {
			if shouldStop() {
				break
			}
			tmp := make([]byte, 1)
			n, err := transmuxer.Read(tmp)
			if n == 0 {
				log.Println("Failed to read byte")
				continue
			}
			if err != nil {
				log.Println(err)
				break
			}
			test = append(test, tmp[0])
			log.Println("Total bytes read so far:", len(test), "(", tmp, ")")
		}
		return nil
	})

	log.Println("Sleeping for", os.Args[1], "seconds...")
	time.Sleep(time.Duration(sleepDur) * time.Second)

	log.Println("Lowering music volume...")
	for i := 100; i >= 5; i-- {
		time.Sleep(10 * time.Millisecond)
		music.SetVolume(float64(i) * 0.01)
	}

	log.Println("Waiting 250ms...")
	time.Sleep(250 * time.Millisecond)

	log.Println("Adding text to speech and waiting...")
	tts, err := transmuxer.AddStreamer("/home/joshuadoes/Desktop/translate_tts.mpga", nil, 1)
	if err != nil {
		panic(err)
	}
	for {
		if !tts.IsRunning() {
			break
		}
		if tts.Err() != nil {
			log.Println(err)
		}

		errFF := make([]byte, 0)
		for {
			tmp := make([]byte, 1)
			n, _ := tts.FFmpeg.ReadError(tmp)
			if n == 0 {
				break
			}
			errFF = append(errFF, tmp[0])
		}
		if len(errFF) > 0 {
			log.Println(string(errFF))
		}
	} //Wait for TTS to finish being read in

	log.Println("Waiting 250ms...")
	time.Sleep(250 * time.Millisecond)

	for i := 5; i <= 100; i++ {
		time.Sleep(10 * time.Millisecond)
		music.SetVolume(float64(i) * 0.01)
	}

	log.Println("Sleeping for 2 more seconds...")
	time.Sleep(2 * time.Second)

	log.Println("Stopping reading task...")
	task.Stop()
	<- task.StopChan() //Wait for it to stop

	log.Println("Closing transmuxer...")
	transmuxer.Close()

	log.Println("Writing", len(test), "bytes to test.mp3...")
	ioutil.WriteFile("test.mp3", test, 0644)
}

