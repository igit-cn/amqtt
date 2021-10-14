# 一个Golang实现的分布式MQTT Broker

## 1.特性
 - 目前仅支持mqtt 3.x 协议
 - 支持websocket协议
 - 支持tls
 - 支持分布式集群
 - 支持eclipse/paho.mqtt.golang、MQTT.fx、MQTTX等客户端

## 2.关于解包模块
目前解包部分使用的是: github.com/eclipse/paho.mqtt.golang/packets

## 3.单机模式

```
docker pull werbenhu/amqtt:latest
docker run -d -p 1884:1884 --name amqtt werbenhu/amqtt 
```

指定配置文件, 关于配置文件，参考 [conf.toml][1]
```
docker run -d -p 1884:1884 -v ./conf.toml:/go/amqtt.toml -e CONFIG="/go/amqtt.toml" --name amqtt werbenhu/amqtt
```

## 4.集群模式
请参考[examples][2]


## 5.性能测试

测试环境：

System: CentOS Linux release 7.6.1810 (Core)

CPU(s): 32, Intel(R) Xeon(R) CPU E5-2650 v2 @ 2.60GHz

MemTotal: 32708532 kB


测试工具：

使用的 [inovex/mqtt-stresser][3]，这里采用的是非集群模式单节点测试，只测试跟mosquitto的对比。


 - 1000个客户端，每客户端发10条消息

`docker run --rm inovex/mqtt-stresser -broker tcp://x.x.x.x:1884 -num-clients 1000 -num-messages 10 -no-progress`

|indicator	| amqtt	| mosquitto |
| ------------- | ------------- | ------------- |
| Pub Fastest	| 34370 msg/sec |	34238 msg/sec |
| Pub Slowest	| 2645 msg/sec |	736 msg/sec |
| Pub Median	| 21729 msg/sec |	22432 msg/sec |
| Rece Fastest	 | 684 msg/sec |	494 msg/sec |
| Rece Slowest	| 330 msg/sec |	146 msg/sec |
| Rece Median |	458 msg/sec	| 190 msg/sec |


 - 10个客户端，每客户端发1000条消息

`docker run --rm inovex/mqtt-stresser -broker tcp://x.x.x.x:1884 -num-clients 10 -num-messages 1000 -no-progress`

| indicator 	| amqtt	| mosquitto |
| ------------- | ------------- | ------------- |
| Pub Fastest	| 53328 msg/sec |	50582 msg/sec |
| Pub Slowest |	24914 msg/sec |	23735 msg/sec |
| Pub Median	| 44282 msg/sec |	41473 msg/sec |
| Rece Fastest |	13446 msg/sec |	3330 msg/sec |
| Rece Slowest |	9670 msg/sec |	3086 msg/sec |
| Rece Median |	11204 msg/sec |	3150 msg/sec |


 - 1个客户端，每客户端发10000条消息

`docker run --rm inovex/mqtt-stresser -broker tcp://x.x.x.x:1884 -num-clients 1 -num-messages 10000 -no-progress`

| indicator	| amqtt	| mosquitto |
| ------------- | ------------- | ------------- |
| Pub Fastest	|28279 msg/sec |	46459 msg/sec |
| Pub Slowest |	28279 msg/sec |	46459 msg/sec |
| Pub Median |	28279 msg/sec |	46459 msg/sec |
| Rece Fastest |	61479 msg/sec |	44560 msg/sec |
| Rece Slowest |	61479 msg/sec |	44560 msg/sec |
| Rece Median |	61479 msg/sec |	44560 msg/sec |
 


  [1]: https://github.com/werbenhu/amqtt/blob/master/conf.toml
  [2]: https://github.com/werbenhu/amqtt/tree/master/example
  [3]: https://github.com/inovex/mqtt-stresser
