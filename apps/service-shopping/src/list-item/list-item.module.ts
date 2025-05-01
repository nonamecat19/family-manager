import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { ListItem } from './entities/list-item.entity';
import { ListItemController } from './list-item.controller';
import { ListItemService } from './list-item.service';
import { CategoryModule } from '../category/category.module';
import { TagModule } from '../tag/tag.module';

@Module({
  imports: [TypeOrmModule.forFeature([ListItem]), CategoryModule, TagModule],
  controllers: [ListItemController],
  providers: [ListItemService],
  exports: [ListItemService],
})
export class ListItemModule {}
