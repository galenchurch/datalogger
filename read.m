%

len = 1000;
ts = len:2;
interval = 0.05;

for v = 1:1:len
    out = writeRead(spi,[hex2dec('00') hex2dec('00')]);
    MSB = bitand(out(1), hex2dec('1F'));

    MSB = cast(MSB, 'uint16');
    MSB = bitshift(MSB,8);
    LSB = cast(out(2), 'uint16');
    val = MSB + LSB;
    data = [v, ((cast(val, 'double') - 4096) /80)]
    ts = [ts; data];
    pause(interval);
end

x = ts(:,1);
y = ts(:,2);
m = movmean(y, 10);
x = x*interval;

M = [x y m];
dt = datestr(now,'yymmdd-HHMMSS');
fn = strcat(dt, '.csv')
csvwrite(fn,M)


hold on
plot(x, y, 'g--', x, m, 'b');
ylim([-2, 2]);
grid;
hold off
