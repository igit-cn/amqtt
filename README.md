# amqtt

mqtt服务器

支持websocket协议

支持tls

支持分布式集群

支持客户端eclipse/paho.mqtt.golang、MQTT.fx、MQTTX


启动单机模式

```
docker pull werbenhu/amqtt:latest
docker run -d -p 1884:1884 --name amqtt werbenhu/amqtt 
```


