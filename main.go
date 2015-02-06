package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os/exec"
	"time"

	"github.com/mesilliac/pulse-simple"
)

func main() {
	var (
		frame       time.Duration
		dBThreshold float64
		cmdStr      string
	)
	flag.DurationVar(&frame, "frame", time.Second, "time frame to capture samples")
	flag.Float64Var(&dBThreshold, "dB", -93.0, "dB threshold to consider it a clap")
	flag.StringVar(&cmdStr, "cmd", "urxvtc", "cmd to launch if threshold is passed")
	flag.Parse()

	ss := pulse.SampleSpec{pulse.SAMPLE_S16LE, 44100, 2}
	stream, err := pulse.Capture("clapclap", "clap stream", &ss)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Free()

	dBCh := make(chan float64)
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
			rms := acc / bufsize
			dBCh <- -20 * math.Log10((1<<16)-math.Sqrt(rms))
		}
		close(dBCh)
	}()

	avgCh := make(chan float64)
	tick := time.Tick(frame)
	go func() {
		var (
			n     int
			dBbuf []float64
		)

		for {
			select {
			case dB := <-dBCh:
				n++
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

	throttle := time.Tick(time.Millisecond * 1000)
	for avg := range avgCh {
		fmt.Println(avg)
		switch {
		case avg < dBThreshold:
			select {
			case <-throttle:
				go func() {
					cmd := exec.Command("sh", "-c", cmdStr)
					if err := cmd.Run(); err != nil {
						log.Println(err)
					}
				}()
			default:
				log.Println("throttled")
			}
		}
	}
}
