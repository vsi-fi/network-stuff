while true; do
battery=`/usr/bin/acpi | awk '{print $3 $4 $5 " " $6 " " $7}'`;
memfreak2=`grep MemFree /proc/meminfo | awk '{ print $2 }'`;
memfreak=$(( $memfreak2/1024 ));
#ssid=`/usr/bin/nmcli -t -f active,ssid dev wifi|grep -oPi '(?:yes:)\K(.*?)$'`;
ssid=`/usr/bin/iw dev wlp2s0 info|grep -oPi 'ssid \K(.*?)$'`;
CLK=$( date +'%a %b %d %R:%S %Z' )
AVG=$( cat /proc/loadavg | cut -d ' ' -f -3 )
xsetroot -name "| Load $AVG | Mem: $memfreak MB free | $battery | SSID: $ssid | $CLK "
sleep 5 
done &

exec /usr/bin/dwm > /dev/null

