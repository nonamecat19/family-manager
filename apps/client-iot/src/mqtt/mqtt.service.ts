import { Injectable, OnModuleInit } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { connect } from 'mqtt'

@Injectable()
export class MqttService implements OnModuleInit {
  private client: any

  constructor(private configService: ConfigService) {}

  onModuleInit() {
    console.log('[MQTT] Connecting to broker...')
    this.client = connect(
      `mqtt://${this.configService.getOrThrow('MQTT_HOST')}:${this.configService.getOrThrow('MQTT_PORT')}`,
    )

    this.client.on('connect', () => {
      console.log('[MQTT] Connected to broker')

      setInterval(() => {
        const temp = (20 + Math.random() * 5).toFixed(2)
        const payload = JSON.stringify({ temp, timestamp: Date.now() })
        this.client.publish('sensor/temperature', payload)
        console.log('[MQTT] Published temperature:', payload)
      }, 5000)

      this.client.subscribe('device/commands')
    })

    this.client.on('message', (topic: any, message: any) => {
      console.log(`[MQTT] Received message on ${topic}: ${message.toString()}`)
    })

    this.client.on('error', console.error)
  }
}
