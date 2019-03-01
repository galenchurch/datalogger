function spi = connect(ip)
    bbb = beaglebone(ip, 'debian', 'temppwd')
    enableSPI(bbb, 0);
    bbb.AvailableSPIChannels;
    spi = spidev(bbb,'spidev1.0', 1);
end
