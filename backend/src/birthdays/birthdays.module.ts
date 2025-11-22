import { Module } from '@nestjs/common';
import { BirthdaysController } from './birthdays.controller';
import { BirthdaysService } from './birthdays.service';
import { DatabaseModule } from '../database/database.module';
import { WebSocketModule } from '../websocket/websocket.module';

@Module({
  imports: [DatabaseModule, WebSocketModule],
  controllers: [BirthdaysController],
  providers: [BirthdaysService],
  exports: [BirthdaysService],
})
export class BirthdaysModule {}


