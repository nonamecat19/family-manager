import { Column, Entity, ManyToMany, PrimaryGeneratedColumn } from 'typeorm';
import { ListItem } from '../../list-item/entities/list-item.entity';

@Entity()
export class Tag {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column({ unique: true })
  name: string;

  @ManyToMany(() => ListItem, (shoppingList) => shoppingList.tags)
  shoppingLists: ListItem[];
}
