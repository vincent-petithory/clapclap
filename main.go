package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/mesilliac/pulse-simple"
)

func main() {
	ss := pulse.SampleSpec{pulse.SAMPLE_S16LE, 44100, 2}
	stream, err := pulse.Capture("clapclap", "clap stream", &ss)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Free()

	rmsCh := make(chan float64)
	go func() {
		const bufsize = 1 << 10
		for {
			data := make([]byte, bufsize)
			if _, err := stream.Read(data); err != nil {
				log.Fatal(err)
			}
			var acc float64
			for i := 0; i < bufsize; i += 2 {
				v := float64(int64(data[i]) + int64(data[i+1])<<8)
				acc += v * v
			}
			rmsCh <- acc / bufsize
		}
		close(rmsCh)
	}()

	avgCh := make(chan float64)
	const frame = time.Millisecond * 1000
	tick := time.Tick(frame)
	go func() {
		var (
			n      int
			rmsbuf []float64
		)

		for {
			select {
			case rms := <-rmsCh:
				n++
				rmsbuf = append(rmsbuf, rms)
			case <-tick:
				var avg float64
				nf := float64(n)
				for _, rms := range rmsbuf {
					avg += rms / nf
				}
				rmsbuf = []float64{}
				n = 0
				dB := -20 * math.Log10(1<<16-math.Sqrt(avg))
				avgCh <- dB
			}
		}
	}()

	for avg := range avgCh {
		fmt.Println(avg)
	}
}
