import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { DatabaseModule } from './database/database.module';
import { CommonModule } from './common/common.module';
import { AuthModule } from './auth/auth.module';
import { FamiliesModule } from './families/families.module';
import { FoldersModule } from './folders/folders.module';
import { ListsModule } from './lists/lists.module';
import { NotesModule } from './notes/notes.module';
import { BirthdaysModule } from './birthdays/birthdays.module';
import { WebSocketModule } from './websocket/websocket.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      envFilePath: '.env',
    }),
    CommonModule,
    DatabaseModule,
    AuthModule,
    FamiliesModule,
    FoldersModule,
    ListsModule,
    NotesModule,
    BirthdaysModule,
    WebSocketModule,
  ],
})
export class AppModule {}

