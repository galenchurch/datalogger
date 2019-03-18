package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
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

func startSPI() {
	out, err := exec.Command("/bin/sh", "./spi_init.sh").Output()
	if err != nil {
		log.Print(err)
	}
	fmt.Printf("CMD: %s\n", out)
}

type conf struct {
	InfluxHost     string `yaml:"influx_host"`
	InfluxDatabase string `yaml:"influx_database"`
	SpiDev         string `yaml:"spi_dev"`
	Mean           int    `yaml:"mean"`
	Interval       int    `yaml:"interval"`
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

	err := l.spiCon.Tx(write, read)
	if err != nil {
		log.Fatal(err)
	}
	d.t = time.Now()
	d.tStamp = d.t.UnixNano()

	d.meas = parseMeasurement(read)
	d.ave = l.moving(d.meas)

	d.loc, err = os.Hostname()
	if err != nil {
		log.Panic("Hostname issue")
	}
}

func (d *DM) fmtToLine() (s string) {
	h, err := os.Hostname()
	if err != nil {
		log.Panic("Hostname issue")
	}
	s = fmt.Sprintf("current,location=%s i=%v,iave_%v=%v %v\n", h, d.meas, d.numAve, d.ave, d.tStamp)

	return s
}

type Logger struct {
	ma       *movingaverage.MovingAverage
	inCon    *client.Client
	spiCon   spi.Conn
	interval int
	cloud    bool
	fp       *os.File
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
	p, err := spireg.Open(cnf.SpiDev)
	if err != nil {
		log.Fatal(err)
	}
	//defer p.Close()

	// Convert the spi.Port into a spi.Conn so it can be used for communication.
	d.spiCon, err = p.Connect(physic.MegaHertz, spi.Mode1, 8)
	if err != nil {
		log.Fatal(err)
	}

	d.infuxOrFile(cnf.InfluxHost)

}

func (d *Logger) moving(a float64) float64 {
	d.ma.Add(a)
	return d.ma.Avg()
}

func (d *Logger) openFile(n string) {
	var err error

	d.fp, err = os.OpenFile(n, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		panic(err)
	}

	if _, err = d.fp.WriteString("DATALOGGER DUMP:\n"); err != nil {
		panic(err)
	}

}

func (d *Logger) writeLine(n int, m float64, a float64) {
	s := fmt.Sprintf("%v,%v,%v\n", n, m, a)
	if _, err := d.fp.WriteString(s); err != nil {
		panic(err)
	}
}

func (d *Logger) infuxOrFile(h string) {

	//helper file function
	of := func() {
		name := genFileName()
		d.openFile(name)
		d.cloud = false
		fmt.Printf("Writing to disk @ %s\n", name)
	}

	//is the host specified as file?
	if h == "file" {
		of()
		return
	}

	//attempt connection to host
	err := d.connInflux(h)
	if err != nil {
		of()
	} else {
		fmt.Printf("Writing to cloud @ %s\n", h)
		d.cloud = true
	}

}

func (d *Logger) connInflux(h string) error {

	host, err := url.Parse(fmt.Sprintf("http://%s:%d", h, 8086))
	if err != nil {
		log.Print(err)
		return err
	}
	d.inCon, err = client.NewClient(client.Config{URL: *host})
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
}

func (d *Logger) writeMeasurement(meas DM) {

	if d.cloud {
		//attempt write to cloud
		err := d.writeToInflux(meas)
		if err == nil {
			return
		} else {
			log.Println("Failled to write to db with cloud = 1")
			d.infuxOrFile("file")
		}
	}
	if _, err := d.fp.WriteString(meas.fmtToLine()); err != nil {
		log.Printf("Write Error: %s", err)
	}
}

func (d *Logger) writeToInflux(meas DM) error {
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
		return err
	}
	return nil
}

func genFileName() string {
	t := time.Now()

	return fmt.Sprintf("%s.txt", t.Format("20060102-150405"))
}

func parseMeasurement(r []byte) float64 {

	scratch := r[0] & 0x1F
	MSB := uint16(scratch) << 8
	val := MSB + uint16(r[1])
	return (float64(val) - 4096) / 80
}

func spitest() {

	na := 10
	dl := Logger{ma: movingaverage.New(na)}

	dl.initFromConf()

	for {

		d := DM{numAve: na}
		d.TLI4970Read(dl)
		dl.writeMeasurement(d)
		time.Sleep(time.Duration(dl.interval) * time.Millisecond)

	}

}

func main() {
	log.Println("DATALOGGER v1\n")
	startSPI()
	log.Println("SPI Configured\n")
	spitest()
}
