import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { ListItem } from './entities/list-item.entity';
import { CreateListItemDto } from './dto/create-list-item.dto';
import { FilterListItemDto } from './dto/filter-list-item.dto';
import { CategoryService } from '../category/category.service';
import { TagService } from '../tag/tag.service';

@Injectable()
export class ListItemService {
  constructor(
    @InjectRepository(ListItem)
    private shoppingListRepository: Repository<ListItem>,
    private categoryService: CategoryService,
    private tagService: TagService,
  ) {}

  async findAll(filterDto?: FilterListItemDto): Promise<ListItem[]> {
    const queryBuilder = this.shoppingListRepository
      .createQueryBuilder('shoppingList')
      .leftJoinAndSelect('shoppingList.category', 'category')
      .leftJoinAndSelect('shoppingList.tags', 'tags');

    if (filterDto?.categoryId) {
      queryBuilder.andWhere('category.id = :categoryId', {
        categoryId: filterDto.categoryId,
      });
    }

    if (filterDto?.tagIds && filterDto.tagIds.length > 0) {
      queryBuilder.andWhere('tags.id IN (:...tagIds)', {
        tagIds: filterDto.tagIds,
      });
    }

    return queryBuilder.getMany();
  }

  async findOne(id: string): Promise<ListItem> {
    const shoppingList = await this.shoppingListRepository.findOne({
      where: { id },
      relations: ['category', 'tags'],
    });

    if (!shoppingList) {
      throw new NotFoundException(`Shopping list with ID ${id} not found`);
    }

    return shoppingList;
  }

  async create(createShoppingListDto: CreateListItemDto): Promise<ListItem> {
    const shoppingList = this.shoppingListRepository.create({
      title: createShoppingListDto.title,
      description: createShoppingListDto.description,
    });

    if (createShoppingListDto.categoryId) {
      shoppingList.category = await this.categoryService.findOne(
        createShoppingListDto.categoryId,
      );
    }

    if (
      createShoppingListDto.tagIds &&
      createShoppingListDto.tagIds.length > 0
    ) {
      shoppingList.tags = await this.tagService.findByIds(
        createShoppingListDto.tagIds,
      );
    }

    return this.shoppingListRepository.save(shoppingList);
  }

  async update(
    id: string,
    updateShoppingListDto: CreateListItemDto,
  ): Promise<ListItem> {
    const shoppingList = await this.findOne(id);

    if (updateShoppingListDto.title) {
      shoppingList.title = updateShoppingListDto.title;
    }

    if (updateShoppingListDto.description !== undefined) {
      shoppingList.description = updateShoppingListDto.description;
    }

    if (updateShoppingListDto.categoryId) {
      shoppingList.category = await this.categoryService.findOne(
        updateShoppingListDto.categoryId,
      );
    }

    if (updateShoppingListDto.tagIds) {
      shoppingList.tags = await this.tagService.findByIds(
        updateShoppingListDto.tagIds,
      );
    }

    return this.shoppingListRepository.save(shoppingList);
  }

  async remove(id: string): Promise<void> {
    const shoppingList = await this.findOne(id);
    await this.shoppingListRepository.remove(shoppingList);
  }
}
