pkill evio
sleep 0.1

./evio -dubbo-port=20887 -mode=p1 -provider-port=30000 &
./evio -dubbo-port=20888 -mode=p2 -provider-port=30001 &
./evio -dubbo-port=20889 -mode=p3 -provider-port=30002 &
./evio


