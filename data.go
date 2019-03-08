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

type DM struct {
	tStamp int64
	meas   float64
	ave    float64
	numAve int
}

func (d *DM) TLI4970Read(c spi.Conn, l dLog) {
	write := []byte{0x00, 0x00}
	read := make([]byte, len(write))
	if err := c.Tx(write, read); err != nil {
		log.Fatal(err)
	}
	t := time.Now()
	d.tStamp = t.UnixNano()
	// Use read.
	fmt.Printf("%v\n", read)
	d.meas = parseMeasurement(read)
	d.ave = l.moving(d.meas)
}

func (d *DM) fmtToLine() (s string) {
	h, err := os.Hostname()
	if err != nil {
		log.Panic("Hostname issue")
	}
	s = fmt.Sprintf("current,location=%s i=%v,iave_%v=%v %v", h, d.meas, d.numAve, d.ave, d.tStamp)
	fmt.Println(s)
	return s
}

func (d *DM) postToDb(host string) {

}

func main() {
	fmt.Printf("test")
	log.Print("Test")
	spitest()
}

type dLog struct {
	ma *movingaverage.MovingAverage
}

func parseMeasurement(r []byte) float64 {

	scratch := r[0] & 0x1F
	MSB := uint16(scratch) << 8
	val := MSB + uint16(r[1])
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
	na := 10
	logger := dLog{ma: movingaverage.New(na)}
	fp := openFile("test.csv")

	for index := 0; index < 60; index++ {

		// write := []byte{0x00, 0x00}
		// read := make([]byte, len(write))
		// if err := c.Tx(write, read); err != nil {
		// 	log.Fatal(err)
		// }
		// t := time.Now()
		// // Use read.
		// fmt.Printf("%v\n", read)

		// meas := parseMeasurement(read)

		// ave := logger.moving(meas)
		// writeLine(fp, index, meas, ave)

		// fmt.Printf("Measurement: %v\n", meas)
		// fmt.Printf("Moving average: %v\n", ave)
		// fmt.Println("Unix nano:", t.UnixNano())
		d := DM{numAve: na}
		d.TLI4970Read(c, logger)
		d.fmtToLine()
		time.Sleep(1000 * time.Millisecond)

	}

	defer fp.Close()

}
