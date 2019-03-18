# datalogger

Datalogger for SPI data on Beaglebone Black.  Currently logs measurements from Infineon TLI4970 Digital Hall Current Sensor: https://www.infineon.com/cms/en/product/sensor/magnetic-current-sensor/tli4970-d050t4/ 

Will post directly to InlfuxDB or fall-back to local file formated with inlfux line-protocol.

### conf.yaml Format

    influx_host: "{localhost | server_ip | file}"
    influx_database: "test"
    spi_dev: "/dev/spidev1.0"
    mean: 10
    interval: 1000
       

build for BBB (ARM)

    env GOOS=linux GOARCH=arm GOARM=7 go build

Enable SPI

    echo BB-SPIDEV0 | sudo tee /sys/devices/bone_capemgr.*/slots