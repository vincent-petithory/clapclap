package main

import (
	"fmt"
	"log"
	"math"
	"time"

	pulsesimple "github.com/mesilliac/pulse-simple"
)

func main() {
	ss := pulsesimple.SampleSpec{pulsesimple.SAMPLE_S16LE, 44100, 2}
	stream, err := pulsesimple.Capture("clapclap", "clap stream", &ss)
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
			n     int
			dBbuf []float64
		)

		for {
			select {
			case rms := <-rmsCh:
				n++
				dB := -20 * math.Log10(1<<16-math.Sqrt(rms))
				dBbuf = append(dBbuf, dB)
			case <-tick:
				var avg float64
				nf := float64(n)
				for _, dB := range dBbuf {
					avg += dB / nf
				}
				dBbuf = []float64{}
				n = 0
				avgCh <- avg
			}
		}
	}()

	for avg := range avgCh {
		fmt.Println(avg)
	}
}
