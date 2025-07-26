import { Controller } from '@nestjs/common'
import { EventPattern, Payload } from '@nestjs/microservices'

@Controller()
export class IotController {
  @EventPattern('sensor/temperature')
  handleTemperature(@Payload() payload: any) {
    console.log(
      `Temperature received: ${payload.temp}°C at ${new Date(payload.time)}`,
    )
  }
}
