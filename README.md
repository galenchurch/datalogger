# datalogger

### conf.yaml Format

    influx_host: "{localhost | server_ip}"
    influx_database: "test"
    spi_dev: "/dev/spidev1.0"
       

build for BBB (ARM)

    env GOOS=linux GOARCH=arm GOARM=7 go build

Enable SPI

    echo BB-SPIDEV0 | sudo tee /sys/devices/bone_capemgr.*/slots