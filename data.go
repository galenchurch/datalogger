package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/conn/spi/spireg"
	"periph.io/x/periph/host"

	movingaverage "github.com/RobinUS2/golang-moving-average"
	client "github.com/influxdata/influxdb1-client"
)

type conf struct {
	Influx_host     string `yaml:"influx_host"`
	Influx_database string `yaml:"influx_database"`
	Spi_dev         string `yaml:"spi_dev"`
	Mean            int    `yaml:"mean"`
	Interval        int    `yaml:"interval"`
}

func (c *conf) getConf() {

	yamlFile, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}

type DM struct {
	t      time.Time
	tStamp int64
	meas   float64
	ave    float64
	numAve int
	loc    string
}

func (d *DM) TLI4970Read(l Logger) {
	write := []byte{0x00, 0x00}
	read := make([]byte, len(write))

	if err := l.spiCon.Tx(write, read); err != nil {
		log.Fatal(err)
	}
	d.t = time.Now()
	d.tStamp = d.t.UnixNano()
	// Use read.
	//fmt.Printf("%v\n", read)
	d.meas = parseMeasurement(read)
	d.ave = l.moving(d.meas)

	d.loc, _ = os.Hostname()
	// if err != nil {
	// 	log.Panic("Hostname issue")
	// }
}

func (d *DM) fmtToLine() (s string) {
	h, err := os.Hostname()
	if err != nil {
		log.Panic("Hostname issue")
	}
	s = fmt.Sprintf("current,location=%s i=%v,iave_%v=%v %v", h, d.meas, d.numAve, d.ave, d.tStamp)
	//fmt.Println(s)
	return s
}

func (d *DM) postToDb(host string) {

}

type Logger struct {
	ma       *movingaverage.MovingAverage
	inCon    *client.Client
	spiCon   spi.Conn
	interval int
}

func (d *Logger) initFromConf() {
	var cnf conf
	cnf.getConf()

	fmt.Println(cnf)
	d.interval = cnf.Interval

	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Use spireg SPI port registry to find the first available SPI bus.
	p, err := spireg.Open(cnf.Spi_dev)
	if err != nil {
		log.Fatal(err)
	}
	//defer p.Close()

	// Convert the spi.Port into a spi.Conn so it can be used for communication.
	d.spiCon, err = p.Connect(physic.MegaHertz, spi.Mode1, 8)
	if err != nil {
		log.Fatal(err)
	}
	//na := 10
	// fp := openFile("test.csv")

	d.connInflux(cnf.Influx_host)

}

func (d *Logger) moving(a float64) float64 {
	d.ma.Add(a)
	return d.ma.Avg()
}

func (d *Logger) connInflux(h string) {

	host, err := url.Parse(fmt.Sprintf("http://%s:%d", h, 8086))
	if err != nil {
		log.Fatal(err)
	}
	d.inCon, err = client.NewClient(client.Config{URL: *host})
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Logger) writeToInflux(meas DM) {
	pts := make([]client.Point, 1)
	pts[0] = client.Point{
		Measurement: "current",
		Tags: map[string]string{
			"location":  meas.loc,
			"num_mean:": strconv.Itoa(meas.numAve),
		},
		Fields: map[string]interface{}{
			"i":    meas.meas,
			"iave": meas.ave,
		},
		Time:      meas.t,
		Precision: "n",
	}

	bps := client.BatchPoints{
		Points:   pts,
		Database: "test",
	}

	_, err := d.inCon.Write(bps)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	fmt.Printf("test")
	log.Print("Test")
	spitest()
}

func parseMeasurement(r []byte) float64 {

	scratch := r[0] & 0x1F
	MSB := uint16(scratch) << 8
	val := MSB + uint16(r[1])
	return (float64(val) - 4096) / 80
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
	// // Make sure periph is initialized.
	// if _, err := host.Init(); err != nil {
	// 	log.Fatal(err)
	// }

	// // Use spireg SPI port registry to find the first available SPI bus.
	// p, err := spireg.Open("/dev/spidev1.0")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer p.Close()

	// // Convert the spi.Port into a spi.Conn so it can be used for communication.
	// c, err := p.Connect(physic.MegaHertz, spi.Mode1, 8)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	na := 10
	dl := Logger{ma: movingaverage.New(na)}
	// fp := openFile("test.csv")

	dl.initFromConf()
	//index := 0; index < 60; index++
	for {

		d := DM{numAve: na}
		d.TLI4970Read(dl)
		//d.fmtToLine()
		dl.writeToInflux(d)
		time.Sleep(time.Duration(dl.interval) * time.Millisecond)

	}

	//defer fp.Close()

}
