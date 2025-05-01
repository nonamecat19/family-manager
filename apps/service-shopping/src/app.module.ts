import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { ListItemModule } from './list-item/list-item.module';
import { DatabaseModule } from './database/database.module';
import { TagModule } from './tag/tag.module';
import { CategoryModule } from './category/category.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
    }),
    DatabaseModule,
    ListItemModule,
    TagModule,
    CategoryModule,
  ],
  controllers: [],
  providers: [],
})
export class AppModule {}
