package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/conn/spi/spireg"
	"periph.io/x/periph/host"

	movingaverage "github.com/RobinUS2/golang-moving-average"
)

func main() {
	fmt.Printf("test")
	log.Print("Test")
	spitest()
}

type dLog struct {
	ma *movingaverage.MovingAverage
}

func parseMeasurement(r []byte) float64 {

	MSB := r[0] & 0x1F
	MSB = MSB << 8
	val := MSB + r[1]
	return (float64(val) - 4096) / 80
}

func (d dLog) moving(a float64) float64 {
	d.ma.Add(a)
	return d.ma.Avg()
}

func openFile(n string) *os.File {

	f, err := os.OpenFile(n, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		panic(err)
	}

	if _, err = f.WriteString("val,val2,val3\n"); err != nil {
		panic(err)
	}

	return f
}
func writeLine(f *os.File, n int, m float64, a float64) {
	s := fmt.Sprintf("%v,%v,%v\n", n, m, a)
	if _, err := f.WriteString(s); err != nil {
		panic(err)
	}
}

func spitest() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Use spireg SPI port registry to find the first available SPI bus.
	p, err := spireg.Open("/dev/spidev1.0")
	if err != nil {
		log.Fatal(err)
	}
	defer p.Close()

	// Convert the spi.Port into a spi.Conn so it can be used for communication.
	c, err := p.Connect(physic.MegaHertz, spi.Mode1, 8)
	if err != nil {
		log.Fatal(err)
	}

	logger := dLog{ma: movingaverage.New(10)}
	fp := openFile("test.csv")

	for index := 0; index < 15; index++ {
		// Write 0x10 to the device, and read a byte right after.
		write := []byte{0x00, 0x00}
		read := make([]byte, len(write))
		if err := c.Tx(write, read); err != nil {
			log.Fatal(err)
		}
		// Use read.
		fmt.Printf("%v\n", read)

		meas := parseMeasurement(read)

		ave := logger.moving(meas)
		writeLine(fp, index, meas, ave)

		fmt.Printf("Measurement: %v\n", meas)
		fmt.Printf("Moving average: %v\n", ave)
		time.Sleep(100 * time.Millisecond)
	}

	defer fp.Close()

}
