/*
MQTT protocol implementaion.

client = new MqttClient("tcp://localhost:1883", "pahomqttpublish1");
options = new MqttConnectOptions();
client.connect(options);
MqttMessage message = new MqttMessage();
message.setPayload("A single message".getBytes());
client.publish("pahodemo/test", message);
client.disconnect();
*/
package mqtt
