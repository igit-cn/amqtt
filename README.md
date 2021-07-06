# Golang写的分布式MQTT Broker

## 1.特性
 - 目前仅支持mqtt 3.x 协议
 - 支持websocket协议
 - 支持tls
 - 支持分布式集群
 - 支持eclipse/paho.mqtt.golang、MQTT.fx、MQTTX等客户端


## 2.单机模式

```
docker pull werbenhu/amqtt:latest
docker run -d -p 1884:1884 --name amqtt werbenhu/amqtt 
```

## 3.集群模式
请参考[examples][1]


  [1]: https://github.com/werbenhu/amqtt/tree/master/example